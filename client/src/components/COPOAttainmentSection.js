import React, { useEffect, useState } from 'react'
import XLSX from 'xlsx-js-style'
import { API_BASE_URL } from '../config'

function formatMarks(marks) {
  if (marks === null || marks === undefined || marks === '') {
    return '-'
  }

  const value = Number(marks)
  if (Number.isNaN(value)) {
    return '-'
  }

  return Number.isInteger(value) ? String(value) : value.toFixed(2)
}

function formatPercentage(value) {
  if (value === null || value === undefined || value === '') {
    return '-'
  }

  const numeric = Number(value)
  if (Number.isNaN(numeric)) {
    return '-'
  }

  return Number.isInteger(numeric) ? String(numeric) : numeric.toFixed(2)
}

function detectTestFamily(name) {
  const normalized = String(name || '').trim().toUpperCase()
  if (!normalized) {
    return ''
  }

  const compact = normalized.replace(/[\s_-]+/g, '')

  if (
    normalized.includes('PERIODICAL TEST 1') ||
    compact.includes('PERIODICALTEST1') ||
    compact.startsWith('PT1')
  ) {
    return 'PT1'
  }

  if (
    normalized.includes('PERIODICAL TEST 2') ||
    compact.includes('PERIODICALTEST2') ||
    compact.startsWith('PT2')
  ) {
    return 'PT2'
  }

  if (
    normalized.includes('INNOVATIVE PRACTICE 1') ||
    compact.includes('INNOVATIVEPRACTICE1') ||
    compact.startsWith('IP1')
  ) {
    return 'IP1'
  }

  if (
    normalized.includes('INNOVATIVE PRACTICE 2') ||
    compact.includes('INNOVATIVEPRACTICE2') ||
    compact.startsWith('IP2')
  ) {
    return 'IP2'
  }

  return ''
}

function getDisplayTestCode(name) {
  const family = detectTestFamily(name)
  if (family) {
    return family
  }

  const normalized = String(name || '').trim().toUpperCase()
  return normalized || 'MARKS'
}

function isIPTestType(name) {
  const normalized = String(name || '').trim().toUpperCase()
  if (!normalized) {
    return false
  }

  const compact = normalized.replace(/[\s_-]+/g, '')
  return normalized.includes('INNOVATIVE PRACTICE') || compact.startsWith('IP')
}

function getColumnMaxMarks(column) {
  const raw = column?.max_marks ?? column?.maxMarks
  const numeric = Number(raw)
  return Number.isNaN(numeric) ? null : numeric
}

function getColumnObtainedMark(row, column) {
  const aggregateKeys = Array.isArray(column?.aggregate_keys) && column.aggregate_keys.length > 0
    ? column.aggregate_keys
    : [column?.key]

  let hasAtLeastOneMark = false
  let total = 0

  aggregateKeys.forEach((key) => {
    const numeric = Number(row?.co_marks?.[key])
    if (!Number.isNaN(numeric)) {
      total += numeric
      hasAtLeastOneMark = true
    }
  })

  return hasAtLeastOneMark ? total : null
}

function getVisibleColumnsForTest(name, allColumns) {
  const family = detectTestFamily(name)
  const columns = Array.isArray(allColumns) ? allColumns : []

  if (isIPTestType(name)) {
    const groups = [
      { key: 'IP1', label: 'IP1', pattern: /(?:INNOVATIVE\s*PRACTICE\s*1|IP\s*1)/i },
      { key: 'IP2', label: 'IP2', pattern: /(?:INNOVATIVE\s*PRACTICE\s*2|IP\s*2)/i }
    ]

    return groups.map((group) => {
      const matching = columns.filter((column) => {
        const sourceName = String(column?.component_name || column?.componentName || column?.label || '').trim()
        return group.pattern.test(sourceName)
      })

      const aggregateKeys = []
      const seenKeys = new Set()
      let totalMaxMarks = 0
      let hasAnyMax = false

      matching.forEach((column) => {
        const key = String(column?.key || '').trim()
        if (key && !seenKeys.has(key)) {
          seenKeys.add(key)
          aggregateKeys.push(key)
        }

        const max = getColumnMaxMarks(column)
        if (max !== null) {
          totalMaxMarks += max
          hasAnyMax = true
        }
      })

      return {
        key: group.key,
        label: group.label,
        aggregate_keys: aggregateKeys,
        max_marks: hasAnyMax ? totalMaxMarks : null,
        maxMarks: hasAnyMax ? totalMaxMarks : null
      }
    })
  }

  if (family === 'PT1' || family === 'PT2') {
    return columns.filter((column) => /^CO\d+$/i.test(String(column?.label || '').trim()))
  }

  return columns
}

function calculateAttainmentPercent(mark, maxMarks) {
  const obtained = Number(mark)
  const maximum = Number(maxMarks)
  if (Number.isNaN(obtained) || Number.isNaN(maximum) || maximum <= 0) {
    return null
  }
  return (obtained / maximum) * 100
}

function COPOAttainmentSection({ course, courseId }) {
  const resolvedCourseId = Number(course?.course_id || course?.id || courseId || 0)
  const resolvedWindowId = Number(
    course?.window?.id ||
    course?.window_id ||
    course?.submitted_expired_window?.id ||
    course?.submitted_expired_window_id ||
    course?.missed_window?.id ||
    course?.missed_window_id ||
    0
  )
  const [testTypes, setTestTypes] = useState([])
  const [selectedTestTypeId, setSelectedTestTypeId] = useState('')
  const [columns, setColumns] = useState([])
  const [poColumns, setPOColumns] = useState([])
  const [poSummary, setPOSummary] = useState([])
  const [rows, setRows] = useState([])
  const [absentCount, setAbsentCount] = useState(0)
  const [searchQuery, setSearchQuery] = useState('')
  const [targetPercentInput, setTargetPercentInput] = useState('')
  const [appliedTargetPercent, setAppliedTargetPercent] = useState(null)
  const [responseTargetPercent, setResponseTargetPercent] = useState(0)
  const [loadingTestTypes, setLoadingTestTypes] = useState(false)
  const [loadingRows, setLoadingRows] = useState(false)
  const [error, setError] = useState('')
  const [targetInputError, setTargetInputError] = useState('')
  const [showScrollTop, setShowScrollTop] = useState(false)
  const [downloadingXlsx, setDownloadingXlsx] = useState(false)

  useEffect(() => {
    if (!resolvedCourseId) {
      setTestTypes([])
      setSelectedTestTypeId('')
      setColumns([])
      setPOColumns([])
      setPOSummary([])
      setRows([])
      setAbsentCount(0)
      setSearchQuery('')
      setTargetPercentInput('')
      setAppliedTargetPercent(null)
      setResponseTargetPercent(0)
      setTargetInputError('')
      return
    }

    const fetchTestTypes = async () => {
      setLoadingTestTypes(true)
      setError('')

      try {
        const response = await fetch(`${API_BASE_URL}/test-types?courseId=${resolvedCourseId}`)
        const data = await response.json()

        if (!response.ok) {
          throw new Error(data.error || 'Failed to load test types')
        }

        setTestTypes(Array.isArray(data) ? data : [])
        setSelectedTestTypeId('')
        setColumns([])
        setPOColumns([])
        setPOSummary([])
        setRows([])
        setAbsentCount(0)
        setSearchQuery('')
        setTargetPercentInput('')
        setAppliedTargetPercent(null)
        setResponseTargetPercent(0)
        setTargetInputError('')
      } catch (err) {
        console.error('Error loading test types:', err)
        setError(err.message || 'Failed to load test types')
        setTestTypes([])
        setSelectedTestTypeId('')
        setColumns([])
        setPOColumns([])
        setPOSummary([])
        setRows([])
        setAbsentCount(0)
        setSearchQuery('')
        setTargetPercentInput('')
        setAppliedTargetPercent(null)
        setResponseTargetPercent(0)
        setTargetInputError('')
      } finally {
        setLoadingTestTypes(false)
      }
    }

    fetchTestTypes()
  }, [resolvedCourseId])

  useEffect(() => {
    if (!resolvedCourseId || !selectedTestTypeId) {
      setColumns([])
      setPOColumns([])
      setPOSummary([])
      setRows([])
      setAbsentCount(0)
      setSearchQuery('')
      setTargetPercentInput('')
      setAppliedTargetPercent(null)
      setResponseTargetPercent(0)
      setTargetInputError('')
      return
    }

    const fetchRows = async () => {
      setLoadingRows(true)
      setError('')

      try {
        const targetPercentQuery = appliedTargetPercent !== null
          ? `&targetPercent=${encodeURIComponent(appliedTargetPercent)}`
          : ''
        const response = await fetch(
          `${API_BASE_URL}/co-po-attainment/students?courseId=${resolvedCourseId}&testTypeId=${selectedTestTypeId}${resolvedWindowId ? `&windowId=${resolvedWindowId}` : ''}${targetPercentQuery}`
        )
        const data = await response.json()

        if (!response.ok) {
          throw new Error(data.error || 'Failed to load CO-PO attainment data')
        }

        setColumns(Array.isArray(data?.columns) ? data.columns : [])
        setPOColumns(Array.isArray(data?.po_columns) ? data.po_columns : [])
        setPOSummary(Array.isArray(data?.po_summary) ? data.po_summary : [])
        const fetchedRows = Array.isArray(data?.students) ? data.students : []
        setRows(fetchedRows)
        setAbsentCount(Number(data?.absent_count) || 0)
        setResponseTargetPercent(Number(data?.target_percent) || 0)
        setSearchQuery('')
        setTargetInputError('')
      } catch (err) {
        console.error('Error loading CO-PO attainment rows:', err)
        setError(err.message || 'Failed to load CO-PO attainment data')
        setColumns([])
        setPOColumns([])
        setPOSummary([])
        setRows([])
        setAbsentCount(0)
        setResponseTargetPercent(0)
        setSearchQuery('')
        setTargetInputError('')
      } finally {
        setLoadingRows(false)
      }
    }

    fetchRows()
  }, [resolvedCourseId, resolvedWindowId, selectedTestTypeId, appliedTargetPercent])

  useEffect(() => {
    const handleScroll = () => {
      setShowScrollTop(window.scrollY > 300)
    }

    window.addEventListener('scroll', handleScroll)
    handleScroll()

    return () => {
      window.removeEventListener('scroll', handleScroll)
    }
  }, [])

  const selectedTest = testTypes.find((item) => String(item.id) === String(selectedTestTypeId))
  const selectedTestFamily = detectTestFamily(selectedTest?.name)
  const isPT1Selected = selectedTestFamily === 'PT1'
  const visibleColumns = getVisibleColumnsForTest(selectedTest?.name, columns)
  const marksColumnLabel = selectedTest ? `${getDisplayTestCode(selectedTest.name)} Marks` : 'Marks'
  const parsedTargetPercent = Number(targetPercentInput)
  const hasValidTargetPercent =
    targetPercentInput.trim() !== '' &&
    !Number.isNaN(parsedTargetPercent) &&
    parsedTargetPercent >= 0 &&
    parsedTargetPercent <= 100
  const poSummaryByKey = poSummary.reduce((accumulator, item) => {
    const key = String(item?.key || '').trim()
    if (key) {
      accumulator[key] = item
    }
    return accumulator
  }, {})
  const orderedPOSummary = poColumns.length > 0
    ? poColumns.map((column) => poSummaryByKey[column.key] || {
      key: column.key,
      label: column.label,
      attainment_percent: 0
    })
    : poSummary
  const filteredRows = rows.filter((row) => {
    const query = searchQuery.trim().toLowerCase()
    if (!query) {
      return true
    }

    return (
      String(row.register_no || '').toLowerCase().includes(query) ||
      String(row.student_name || '').toLowerCase().includes(query)
    )
  })
  const sortedFilteredRows = [...filteredRows].sort((left, right) => {
    const leftRegister = String(left.register_no || '').trim()
    const rightRegister = String(right.register_no || '').trim()
    if (leftRegister !== rightRegister) {
      return leftRegister.localeCompare(rightRegister)
    }
    return String(left.student_name || '').localeCompare(String(right.student_name || ''))
  })
  const handleScrollToTop = () => {
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  const resetCOPOViewState = () => {
    setColumns([])
    setPOColumns([])
    setPOSummary([])
    setRows([])
    setAbsentCount(0)
    setSearchQuery('')
    setTargetPercentInput('')
    setAppliedTargetPercent(null)
    setResponseTargetPercent(0)
    setTargetInputError('')
    setError('')
  }

  const handleTestTypeChange = (event) => {
    const nextTestTypeId = event.target.value
    resetCOPOViewState()
    setSelectedTestTypeId(nextTestTypeId)
  }

  const handleGenerateAttainment = () => {
    if (!hasValidTargetPercent) {
      setTargetInputError('Enter a valid target CO % between 0 and 100.')
      return
    }

    setAppliedTargetPercent(parsedTargetPercent)
    setTargetInputError('')
  }

  const summaryColumns = visibleColumns
  const attendedStudentsCount = rows.length
  const totalStudentsCount = rows.length + absentCount
  const courseCode = String(course?.course_code || '-').trim() || '-'
  const courseName = String(course?.course_name || 'Course context unavailable').trim() || 'Course context unavailable'
  const showGeneratedSummary =
    appliedTargetPercent !== null &&
    selectedTestTypeId &&
    summaryColumns.length > 0 &&
    rows.length > 0

  const attainmentLevelByPercent = (attainmentPercent) => {
    const value = Number(attainmentPercent)
    if (Number.isNaN(value) || value < 50) return 0
    if (value < 60) return 1
    if (value < 70) return 2
    return 3
  }

  const coSummaryRows = summaryColumns.map((column) => {
    const achievedCount = rows.reduce((count, row) => {
      const percentage = calculateAttainmentPercent(getColumnObtainedMark(row, column), getColumnMaxMarks(column))
      if (percentage === null || appliedTargetPercent === null) {
        return count
      }
      return percentage >= appliedTargetPercent ? count + 1 : count
    }, 0)

    const attainmentPercent = attendedStudentsCount > 0
      ? (achievedCount / attendedStudentsCount) * 100
      : 0

    return {
      key: column.key,
      label: column.label,
      totalStudents: attendedStudentsCount,
      achievedCount,
      attainmentLevel: attainmentLevelByPercent(attainmentPercent),
      remedialCount: attendedStudentsCount - achievedCount
    }
  })

  const sanitizeFileNamePart = (value, fallback) => {
    const cleaned = String(value || '')
      .trim()
      .replace(/[^a-zA-Z0-9_-]+/g, '_')
      .replace(/^_+|_+$/g, '')
    return cleaned || fallback
  }

  const handleDownloadAttainmentXlsx = async () => {
    if (!showGeneratedSummary || !selectedTestTypeId || visibleColumns.length === 0) {
      return
    }

    try {
      setDownloadingXlsx(true)
      setError('')

      const family = detectTestFamily(selectedTest?.name)
      if (family !== 'PT1') {
        setError('Template download is currently available only for Periodical Test 1 (PT1).')
        return
      }

      const targetPercentForDownload = appliedTargetPercent !== null
        ? appliedTargetPercent
        : responseTargetPercent
      const downloadTargetPercentQuery = targetPercentForDownload !== null && targetPercentForDownload !== undefined
        ? `&targetPercent=${encodeURIComponent(targetPercentForDownload)}`
        : ''
      const latestResponse = await fetch(
        `${API_BASE_URL}/co-po-attainment/students?courseId=${resolvedCourseId}&testTypeId=${selectedTestTypeId}${resolvedWindowId ? `&windowId=${resolvedWindowId}` : ''}${downloadTargetPercentQuery}`
      )
      const latestData = await latestResponse.json()
      if (!latestResponse.ok) {
        throw new Error(latestData.error || 'Failed to refresh CO-PO attainment data for download')
      }

      const downloadRows = Array.isArray(latestData?.students) ? latestData.students : []
      const downloadColumns = Array.isArray(latestData?.columns) ? latestData.columns : columns
      const downloadVisibleColumns = getVisibleColumnsForTest(selectedTest?.name, downloadColumns)
      const downloadAbsentCount = Number(latestData?.absent_count) || 0
      const downloadResponseTargetPercent = Number(latestData?.target_percent) || 0
      const downloadAttendedStudentsCount = downloadRows.length
      const downloadTotalStudentsCount = downloadAttendedStudentsCount + downloadAbsentCount
      const downloadCoSummaryRows = downloadVisibleColumns.map((column) => {
        const achievedCount = downloadRows.reduce((count, row) => {
          const percentage = calculateAttainmentPercent(getColumnObtainedMark(row, column), getColumnMaxMarks(column))
          if (percentage === null || targetPercentForDownload === null) {
            return count
          }
          return percentage >= targetPercentForDownload ? count + 1 : count
        }, 0)

        const attainmentPercent = downloadAttendedStudentsCount > 0
          ? (achievedCount / downloadAttendedStudentsCount) * 100
          : 0

        return {
          key: column.key,
          label: column.label,
          totalStudents: downloadAttendedStudentsCount,
          achievedCount,
          attainmentLevel: attainmentLevelByPercent(attainmentPercent),
          remedialCount: downloadAttendedStudentsCount - achievedCount
        }
      })

      setColumns(downloadColumns)
      setRows(downloadRows)
      setAbsentCount(downloadAbsentCount)
      setResponseTargetPercent(downloadResponseTargetPercent)

      const templateUrl = `${process.env.PUBLIC_URL || ''}/templates/22EE401_template.xlsx`
      const templateResponse = await fetch(templateUrl)
      if (!templateResponse.ok) {
        throw new Error('Unable to load XLSX template file')
      }
      const templateBuffer = await templateResponse.arrayBuffer()
      const workbook = XLSX.read(templateBuffer, { type: 'array', cellStyles: true })

      const sheetName = 'Test 1'
      const ws = workbook.Sheets[sheetName]
      if (!ws) {
        throw new Error(`Template sheet "${sheetName}" not found`)
      }

      const slotCount = 5
      const slotLabels = Array.from({ length: slotCount }, (_, index) => `CO${index + 1}`)
      const slotColumns = Array.from({ length: slotCount }, () => null)
      const coSummaryByKey = coSummaryRows.reduce((accumulator, item) => {
        accumulator[item.key] = item
        return accumulator
      }, {})

      const assignColumnToSlot = (column) => {
        const normalizedLabel = String(column?.label || '').trim().toUpperCase()
        const coMatch = normalizedLabel.match(/^CO\s*([1-5])$/i)
        if (coMatch) {
          const coIndex = Number(coMatch[1]) - 1
          if (coIndex >= 0 && coIndex < slotCount && !slotColumns[coIndex]) {
            slotColumns[coIndex] = column
            slotLabels[coIndex] = `CO${coIndex + 1}`
            return
          }
        }

        const emptyIndex = slotColumns.findIndex((item) => item === null)
        if (emptyIndex >= 0) {
          slotColumns[emptyIndex] = column
          slotLabels[emptyIndex] = String(column?.label || slotLabels[emptyIndex])
        }
      }
      visibleColumns.forEach(assignColumnToSlot)

      const getSlotSummaryValues = (slotIndex) => {
        const column = slotColumns[slotIndex]
        if (!column) {
          return {
            totalStudents: 0,
            achievedCount: 0,
            attainmentLevel: '',
            remedialCount: 0
          }
        }
        return coSummaryByKey[column.key] || {
          totalStudents: attendedStudentsCount,
          achievedCount: 0,
          attainmentLevel: '',
          remedialCount: attendedStudentsCount
        }
      }

      const getCellAddress = (row, column) => XLSX.utils.encode_cell({ r: row - 1, c: column - 1 })
      const setWorksheetRange = (lastRow, lastColumn) => {
        ws['!ref'] = XLSX.utils.encode_range({
          s: { r: 0, c: 0 },
          e: { r: lastRow - 1, c: lastColumn - 1 }
        })
      }
      const mergeCells = (rowStart, colStart, rowEnd, colEnd) => {
        ws['!merges'] = Array.isArray(ws['!merges']) ? ws['!merges'] : []
        const nextMerge = {
          s: { r: rowStart - 1, c: colStart - 1 },
          e: { r: rowEnd - 1, c: colEnd - 1 }
        }
        const alreadyMerged = ws['!merges'].some((merge) =>
          merge.s.r === nextMerge.s.r &&
          merge.s.c === nextMerge.s.c &&
          merge.e.r === nextMerge.e.r &&
          merge.e.c === nextMerge.e.c
        )
        if (!alreadyMerged) {
          ws['!merges'].push(nextMerge)
        }
      }
      const cloneCellStyle = (style) => {
        if (!style) {
          return undefined
        }
        return JSON.parse(JSON.stringify(style))
      }
      const ensureCellFromTemplate = (row, column, sourceRow = 16) => {
        const address = getCellAddress(row, column)
        if (ws[address]) {
          return ws[address]
        }

        const sourceAddress = getCellAddress(sourceRow, column)
        const sourceCell = ws[sourceAddress] || {}
        const createdCell = {
          t: sourceCell.t || 's',
          v: '',
          ...(sourceCell.z ? { z: sourceCell.z } : {}),
          ...(sourceCell.s ? { s: cloneCellStyle(sourceCell.s) } : {})
        }
        ws[address] = createdCell
        return createdCell
      }
      const ensureDataRowStyles = (rowStart, rowEnd) => {
        for (let row = rowStart; row <= rowEnd; row += 1) {
          for (let col = 1; col <= 23; col += 1) {
            ensureCellFromTemplate(row, col, 16)
          }
        }
      }
      const getCellValue = (row, column) => {
        const cell = ws[getCellAddress(row, column)]
        return cell?.v ?? ''
      }
      const setCellValue = (row, column, value) => {
        const address = getCellAddress(row, column)
        const existing = ensureCellFromTemplate(row, column, row >= 16 ? 16 : row) || {}
        delete existing.f
        delete existing.F
        if (value === '' || value === null || value === undefined) {
          existing.v = ''
          existing.t = 's'
          delete existing.w
          ws[address] = existing
          return
        }
        if (typeof value === 'number' && Number.isFinite(value)) {
          existing.v = value
          existing.t = 'n'
          delete existing.w
          ws[address] = existing
          return
        }
        existing.v = String(value)
        existing.t = 's'
        delete existing.w
        ws[address] = existing
      }
      const clearRangeValues = (rowStart, colStart, rowEnd, colEnd) => {
        for (let row = rowStart; row <= rowEnd; row += 1) {
          for (let col = colStart; col <= colEnd; col += 1) {
            const address = getCellAddress(row, col)
            if (ws[address]) {
              delete ws[address].f
              delete ws[address].F
              ws[address].v = ''
              ws[address].t = 's'
              delete ws[address].w
            }
          }
        }
      }
      const removeAllFormulas = () => {
        Object.keys(ws).forEach((address) => {
          if (address.startsWith('!')) {
            return
          }

          const cell = ws[address]
          if (!cell) {
            return
          }

          delete cell.f
          delete cell.F
          delete cell.D
          delete cell.si

          if (cell.t === 'e') {
            const formatted = String(cell.w || '')
            const raw = typeof cell.v === 'string' ? cell.v : ''
            if (formatted.includes('#REF!') || raw.includes('#REF!') || cell.v === 0x17) {
              cell.t = 's'
              cell.v = ''
              delete cell.w
            }
          }
        })
      }
      const numericOrBlank = (value) => {
        const numeric = Number(value)
        return Number.isFinite(numeric) ? numeric : ''
      }
      const border = (style = 'thin') => ({
        top: { style, color: { rgb: 'FF000000' } },
        bottom: { style, color: { rgb: 'FF000000' } },
        left: { style, color: { rgb: 'FF000000' } },
        right: { style, color: { rgb: 'FF000000' } }
      })
      const centered = { horizontal: 'center', vertical: 'center', wrapText: true }
      const fill = (rgb) => ({ patternType: 'solid', fgColor: { rgb }, bgColor: { rgb } })
      const applyCellStyle = (row, column, style) => {
        const cell = ensureCellFromTemplate(row, column, row >= 16 ? 16 : row)
        cell.s = {
          ...(cell.s || {}),
          ...style
        }
      }
      const applyRangeStyle = (rowStart, colStart, rowEnd, colEnd, style) => {
        for (let row = rowStart; row <= rowEnd; row += 1) {
          for (let col = colStart; col <= colEnd; col += 1) {
            applyCellStyle(row, col, style)
          }
        }
      }
      const applyCellBorder = (row, column, borderPatch) => {
        const cell = ensureCellFromTemplate(row, column, row >= 16 ? 16 : row)
        cell.s = {
          ...(cell.s || {}),
          border: {
            ...((cell.s && cell.s.border) || {}),
            ...borderPatch
          }
        }
      }
      const applyOuterBorder = (rowStart, colStart, rowEnd, colEnd, style = 'medium') => {
        const line = { style, color: { rgb: 'FF000000' } }
        for (let col = colStart; col <= colEnd; col += 1) {
          applyCellBorder(rowStart, col, { top: line })
          applyCellBorder(rowEnd, col, { bottom: line })
        }
        for (let row = rowStart; row <= rowEnd; row += 1) {
          applyCellBorder(row, colStart, { left: line })
          applyCellBorder(row, colEnd, { right: line })
        }
      }
      const applyHorizontalBorder = (row, colStart, colEnd, side, style = 'medium') => {
        const line = { style, color: { rgb: 'FF000000' } }
        for (let col = colStart; col <= colEnd; col += 1) {
          applyCellBorder(row, col, { [side]: line })
        }
      }
      const applyVerticalDivider = (rowStart, rowEnd, leftCol, rightCol, style = 'medium') => {
        const line = { style, color: { rgb: 'FF000000' } }
        for (let row = rowStart; row <= rowEnd; row += 1) {
          applyCellBorder(row, leftCol, { right: line })
          applyCellBorder(row, rightCol, { left: line })
        }
      }
      const applyTemplateStyles = (lastRow) => {
        const titleStyle = {
          font: { name: 'Calibri', sz: 20, bold: true, color: { rgb: 'FF000000' } },
          alignment: centered
        }
        const subtitleStyle = {
          font: { name: 'Calibri', sz: 15, bold: true, color: { rgb: 'FF000000' } },
          alignment: centered
        }
        const labelStyle = {
          font: { name: 'Calibri', sz: 12, bold: true, color: { rgb: 'FF000000' } },
          alignment: { horizontal: 'right', vertical: 'center', wrapText: true },
          border: border('thin')
        }
        const valueStyle = {
          font: { name: 'Calibri', sz: 12, bold: true, color: { rgb: 'FF000000' } },
          alignment: centered,
          border: border('thin'),
          fill: fill('FFFFFFFF')
        }
        const testTitleStyle = {
          font: { name: 'Calibri', sz: 14, bold: true, underline: true, color: { rgb: 'FF000000' } },
          alignment: centered,
          border: border('medium'),
          fill: fill('FFFFFFFF')
        }
        const summaryStyle = {
          font: { name: 'Calibri', sz: 12, bold: true, color: { rgb: 'FF000000' } },
          alignment: centered,
          border: border('thin'),
          fill: fill('FFFFFFFF')
        }
        const summaryInstructionStyle = {
          ...summaryStyle,
          alignment: { horizontal: 'left', vertical: 'center', wrapText: true }
        }
        const summaryTitleStyle = {
          ...summaryStyle,
          alignment: centered
        }
        const summaryHighlightStyle = {
          ...summaryStyle,
          fill: fill('FFFFFF00')
        }
        const tableHeaderStyle = {
          font: { name: 'Calibri', sz: 12, bold: true, color: { rgb: 'FF000000' } },
          alignment: centered,
          border: border('thin'),
          fill: fill('FF7AD592')
        }
        const dataStyle = {
          font: { name: 'Calibri', sz: 12, color: { rgb: 'FF000000' } },
          alignment: centered,
          border: border('thin')
        }
        const dataWhiteStyle = {
          ...dataStyle,
          fill: fill('FFFFFFFF')
        }

        applyRangeStyle(1, 1, 1, 23, titleStyle)
        applyRangeStyle(2, 1, 2, 23, subtitleStyle)
        applyRangeStyle(3, 1, 3, 23, subtitleStyle)
        applyRangeStyle(4, 1, 4, 19, labelStyle)
        applyRangeStyle(4, 3, 4, 16, valueStyle)
        applyRangeStyle(4, 18, 4, 19, valueStyle)
        applyRangeStyle(6, 1, 6, 23, testTitleStyle)
        applyRangeStyle(7, 1, 12, 22, summaryStyle)
        applyRangeStyle(7, 2, 9, 11, summaryInstructionStyle)
        applyRangeStyle(7, 12, 7, 22, summaryTitleStyle)
        applyRangeStyle(7, 1, 9, 1, summaryTitleStyle)
        applyRangeStyle(11, 18, 11, 22, summaryHighlightStyle)
        applyRangeStyle(14, 1, 15, 23, tableHeaderStyle)
        applyRangeStyle(16, 1, lastRow, 23, dataStyle)
        applyRangeStyle(16, 3, lastRow, 7, dataWhiteStyle)
        applyRangeStyle(16, 13, lastRow, 17, dataWhiteStyle)
        applyOuterBorder(4, 1, 4, 19)
        applyOuterBorder(6, 1, 6, 23)
        applyOuterBorder(7, 1, 12, 22, 'thick')
        applyOuterBorder(7, 1, 12, 11, 'thick')
        applyOuterBorder(7, 12, 12, 22, 'thick')
        applyVerticalDivider(7, 12, 11, 12, 'thick')
        applyHorizontalBorder(7, 1, 22, 'top', 'thick')
        applyHorizontalBorder(12, 1, 22, 'bottom', 'thick')
        applyOuterBorder(14, 1, lastRow, 23)
        applyOuterBorder(10, 18, 11, 22)

        ws['!rows'] = ws['!rows'] || []
        ws['!rows'][0] = { ...(ws['!rows'][0] || {}), hpt: 28 }
        ws['!rows'][1] = { ...(ws['!rows'][1] || {}), hpt: 24 }
        ws['!rows'][2] = { ...(ws['!rows'][2] || {}), hpt: 34 }
        ws['!rows'][5] = { ...(ws['!rows'][5] || {}), hpt: 28 }
        ws['!rows'][6] = { ...(ws['!rows'][6] || {}), hpt: 36 }
        ws['!rows'][7] = { ...(ws['!rows'][7] || {}), hpt: 22 }
        ws['!rows'][8] = { ...(ws['!rows'][8] || {}), hpt: 22 }
        ws['!rows'][13] = { ...(ws['!rows'][13] || {}), hpt: 22 }
        ws['!rows'][14] = { ...(ws['!rows'][14] || {}), hpt: 22 }
      }
      const targetUsed = appliedTargetPercent === null
        ? numericOrBlank(responseTargetPercent)
        : numericOrBlank(appliedTargetPercent)
      const testHeading = (() => {
        if (family === 'PT1') return 'PT 1'
        if (family === 'PT2') return 'PT 2'
        if (family === 'IP1') return 'IP1'
        if (family === 'IP2') return 'IP2'
        return String(selectedTest?.name || 'TEST').toUpperCase()
      })()
      const marksHeaderLabel = getDisplayTestCode(selectedTest?.name)
      const exportRows = [...rows].sort((left, right) => {
        const leftRegister = String(left.register_no || '').trim()
        const rightRegister = String(right.register_no || '').trim()
        if (leftRegister !== rightRegister) {
          return leftRegister.localeCompare(rightRegister)
        }
        return String(left.student_name || '').localeCompare(String(right.student_name || ''))
      })
      const lastDataRow = Math.max(16, 15 + exportRows.length)

      // Keep template layout and styling as-is, update only values.
      setCellValue(1, 1, 'BANNARI AMMAN INSTITUTE OF TECHNOLOGY')
      setCellValue(
        2,
        1,
        'An Autonomous Institution Affiliated to Anna University - Chennai - Approved by AICTE - Accredited by NAAC with "A+" Grade'
      )
      setCellValue(3, 1, '\nSATHYAMANGALAM - 638401   ERODE DISTRICT   TAMILNADU   INDIA')
      mergeCells(3, 1, 3, 23)
      clearRangeValues(3, 2, 3, 23)
      clearRangeValues(8, 18, 12, 22)
      clearRangeValues(14, 3, 15, 23)
      clearRangeValues(16, 1, 600, 23)
      ensureDataRowStyles(16, lastDataRow)

      setCellValue(4, 3, String(course?.academic_year || getCellValue(4, 3)))
      setCellValue(4, 7, String(course?.semester_display || course?.semester_name || getCellValue(4, 7)))
      setCellValue(4, 11, `${courseCode} & ${courseName}`)
      setCellValue(4, 18, '')
      setCellValue(4, 19, '')
      setCellValue(6, 1, testHeading)

      setCellValue(10, 5, targetUsed)
      setCellValue(11, 5, numericOrBlank(totalStudentsCount))
      setCellValue(12, 5, numericOrBlank(attendedStudentsCount))

      for (let slotIndex = 0; slotIndex < slotCount; slotIndex += 1) {
        const summary = getSlotSummaryValues(slotIndex)
        const targetColumn = 18 + slotIndex
        setCellValue(8, targetColumn, slotLabels[slotIndex])
        setCellValue(9, targetColumn, numericOrBlank(summary.totalStudents))
        setCellValue(10, targetColumn, numericOrBlank(summary.achievedCount))
        setCellValue(11, targetColumn, numericOrBlank(summary.attainmentLevel))
        setCellValue(12, targetColumn, numericOrBlank(summary.remedialCount))
      }

      setCellValue(14, 1, 'S.No')
      setCellValue(14, 2, 'Reg.No.')
      setCellValue(14, 3, 'MARKS ALLOCATED')
      setCellValue(14, 8, 'MARKS OBTAINED')
      setCellValue(14, 13, 'CO ATTAINMENT %')
      setCellValue(14, 18, 'ATTAINMENT of COs')
      setCellValue(14, 23, marksHeaderLabel)
      setCellValue(15, 23, 'Marks')

      for (let slotIndex = 0; slotIndex < slotCount; slotIndex += 1) {
        const label = slotLabels[slotIndex]
        setCellValue(15, 3 + slotIndex, label)
        setCellValue(15, 8 + slotIndex, label)
        setCellValue(15, 13 + slotIndex, label)
        setCellValue(15, 18 + slotIndex, label)
      }

      exportRows.forEach((row, index) => {
        const rowNumber = 16 + index
        setCellValue(rowNumber, 1, index + 1)
        setCellValue(rowNumber, 2, row.register_no || '-')

        for (let slotIndex = 0; slotIndex < slotCount; slotIndex += 1) {
          const column = slotColumns[slotIndex]
          if (!column) {
            continue
          }

          const maxMarks = getColumnMaxMarks(column)
          const obtainedMark = getColumnObtainedMark(row, column)
          const percentage = calculateAttainmentPercent(obtainedMark, maxMarks)
          const isCompared = appliedTargetPercent !== null && percentage !== null
          const isAchieved = isCompared && percentage >= appliedTargetPercent

          setCellValue(rowNumber, 3 + slotIndex, numericOrBlank(maxMarks))
          setCellValue(rowNumber, 8 + slotIndex, numericOrBlank(obtainedMark))
          setCellValue(rowNumber, 13 + slotIndex, numericOrBlank(percentage))
          setCellValue(rowNumber, 18 + slotIndex, isCompared ? (isAchieved ? 1 : 0) : '')
        }

        const totalMarks = visibleColumns.reduce((sum, column) => {
          const mark = getColumnObtainedMark(row, column)
          return mark === null ? sum : sum + mark
        }, 0)
        setCellValue(rowNumber, 23, numericOrBlank(totalMarks))
      })

      applyTemplateStyles(lastDataRow)
      removeAllFormulas()
      setWorksheetRange(lastDataRow, 23)
      delete workbook.CalcChain

      const safeCourse = sanitizeFileNamePart(courseCode, 'course')
      const pt1SheetName = 'PT 1'
      workbook.SheetNames = [pt1SheetName]
      workbook.Sheets = { [pt1SheetName]: ws }
      if (workbook.Workbook && Array.isArray(workbook.Workbook.Sheets)) {
        workbook.Workbook.Sheets = [{ name: pt1SheetName }]
      }
      XLSX.writeFile(workbook, `co_po_attainment_${safeCourse}_PT1.xlsx`)
    } catch (downloadError) {
      console.error('Error downloading attainment XLSX:', downloadError)
      setError(downloadError?.message || 'Failed to download XLSX')
    } finally {
      setDownloadingXlsx(false)
    }
  }

  const sectionCardClass = 'rounded-2xl border border-gray-400 bg-white shadow-sm'
  const attainmentSectionCardClass = 'rounded-2xl border border-gray-400 bg-white shadow-sm'
  const attainmentSectionHeaderClass = 'border-b border-gray-400 px-5 py-5'
  const sectionHeaderClass = 'border-b border-gray-400 bg-gray-50 px-5 py-4'
  const metricCardClass = 'rounded-xl border border-gray-400 bg-white p-4 shadow-sm'
  const topStatCardClass = 'min-h-[102px] rounded-xl border border-gray-400 bg-white p-4 shadow-sm flex flex-col justify-between'
  const subtleMetricCardClass = 'rounded-xl border border-gray-400 bg-gray-50 p-4'
  const fieldClass = 'h-11 w-full rounded-lg border border-gray-300 bg-white px-3.5 text-sm text-gray-900 shadow-sm transition focus:border-purple-500 focus:outline-none focus:ring-4 focus:ring-purple-100 disabled:bg-gray-50 disabled:text-gray-400'
  const emptyStateCardClass = `${sectionCardClass} p-10 text-center`
  const statLabelClass = 'text-xs font-medium uppercase tracking-[0.08em] text-gray-500'
  const statValueClass = 'text-base font-semibold text-gray-900'

  return (
    <div className="space-y-6">
      <div className={attainmentSectionCardClass}>
        <div className={attainmentSectionHeaderClass}>
          <h3 className="text-xl font-semibold text-gray-900">CO-PO Attainment</h3>
          <p className="mt-1.5 max-w-3xl text-sm leading-6 text-gray-600">
            View the selected test in an Excel-style CO-wise layout across all mapped teachers.
          </p>
        </div>

        <div className="space-y-4 p-5">
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
            <div className={topStatCardClass}>
              <label htmlFor="testType" className={statLabelClass}>
                Test Type
              </label>
              <select
                id="testType"
                value={selectedTestTypeId}
                onChange={handleTestTypeChange}
                disabled={loadingTestTypes || !testTypes.length}
                className={fieldClass}
              >
                <option value="">Select a test type</option>
                {testTypes.map((testType) => (
                  <option key={testType.id} value={testType.id}>
                    {testType.name}
                  </option>
                ))}
              </select>
            </div>

            <div className={topStatCardClass}>
              <p className={statLabelClass}>Course Code</p>
              <p className="truncate text-lg font-semibold text-gray-900">{courseCode}</p>
            </div>
            <div className={topStatCardClass}>
              <p className={statLabelClass}>Course Name</p>
              <p className="truncate text-base font-semibold text-gray-900">{courseName}</p>
            </div>
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
            <div className={topStatCardClass}>
              <p className={statLabelClass}>Present Students</p>
              <p className={statValueClass}>{rows.length}</p>
            </div>
            <div className={topStatCardClass}>
              <p className={statLabelClass}>Absentees</p>
              <p className={statValueClass}>{absentCount}</p>
            </div>
            <div className={topStatCardClass}>
              <p className={statLabelClass}>Selected Test</p>
              <p className="truncate text-base font-semibold text-gray-900">
                {selectedTest?.name || '-'}
              </p>
            </div>
            <div className={topStatCardClass}>
              <p className={statLabelClass}>Target Used</p>
              <p className={statValueClass}>
                {appliedTargetPercent === null ? formatPercentage(responseTargetPercent) : formatPercentage(appliedTargetPercent)}
              </p>
            </div>
          </div>
        </div>
      </div>

      <div className={`${sectionCardClass} p-5`}>
        <div className="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
          <div>
            <label htmlFor="targetPercent" className="block text-sm font-semibold text-gray-800">
              Target CO %
            </label>
            <p className="mt-1 text-sm text-gray-500">
              Set the benchmark percentage and generate attainment analytics.
            </p>
          </div>

          <div className="w-full xl:max-w-2xl">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-end">
              <input
                id="targetPercent"
                type="number"
                min="0"
                max="100"
                step="0.01"
                value={targetPercentInput}
                onChange={(event) => setTargetPercentInput(event.target.value)}
                placeholder="Enter target CO %"
                className={`${fieldClass} sm:max-w-xs`}
              />
              <button
                type="button"
                onClick={handleGenerateAttainment}
                disabled={!selectedTestTypeId || loadingTestTypes || loadingRows || !hasValidTargetPercent}
                className="inline-flex h-12 items-center justify-center gap-2 rounded-lg bg-purple-500 px-6 text-sm font-semibold text-white shadow-md transition hover:bg-purple-600 disabled:cursor-not-allowed disabled:bg-gray-300 disabled:shadow-none"
              >
                <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                </svg>
                Generate CO-PO Attainment
              </button>
            </div>
          </div>
        </div>
        {targetInputError && (
          <p className="mt-3 text-sm font-medium text-red-600">{targetInputError}</p>
        )}
      </div>

      {showGeneratedSummary && (
        <div className={`${sectionCardClass} overflow-hidden`}>
          <div className={sectionHeaderClass}>
            <h4 className="text-base font-semibold text-gray-900">
              {getDisplayTestCode(selectedTest?.name)} - CO Attainment Summary
            </h4>
          </div>

          <div className="space-y-4 p-5">
            <div className="overflow-hidden rounded-xl border border-gray-400 bg-white">
              <div className="grid grid-cols-1 lg:grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)]">
                <div className="border-b border-gray-400 bg-gray-50 p-5 lg:border-b-0 lg:border-r">
                  <p className="text-sm font-semibold text-gray-900">CO Attainment Levels</p>
                  <div className="mt-3 space-y-2.5 text-sm text-gray-700">
                    <div className="flex items-start gap-2">
                      <span className="mt-1.5 h-1.5 w-1.5 rounded-full bg-gray-500" />
                      <p><span className="font-semibold text-gray-900">Level 0</span> → &lt;50% of students scoring more than the target marks</p>
                    </div>
                    <hr className="border-gray-400" />
                    <div className="flex items-start gap-2">
                      <span className="mt-1.5 h-1.5 w-1.5 rounded-full bg-purple-500" />
                      <p><span className="font-semibold text-gray-900">Level 1</span> → 50% of students scoring more than the target marks</p>
                    </div>
                    <hr className="border-gray-400" />
                    <div className="flex items-start gap-2">
                      <span className="mt-1.5 h-1.5 w-1.5 rounded-full bg-purple-500" />
                      <p><span className="font-semibold text-gray-900">Level 2</span> → 60% of students scoring more than the target marks</p>
                    </div>
                    <hr className="border-gray-400" />
                    <div className="flex items-start gap-2">
                      <span className="mt-1.5 h-1.5 w-1.5 rounded-full bg-purple-500" />
                      <p><span className="font-semibold text-gray-900">Level 3</span> → 70% of students scoring more than the target marks</p>
                    </div>
                  </div>
                </div>

                <div className="p-5">
                  <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
                    <div className={subtleMetricCardClass}>
                      <p className={statLabelClass}>Target CO%</p>
                      <p className="mt-1 text-base font-bold text-gray-900">{formatPercentage(appliedTargetPercent)}</p>
                    </div>
                    <div className={subtleMetricCardClass}>
                      <p className={statLabelClass}>Total No. of Students</p>
                      <p className="mt-1 text-base font-bold text-gray-900">{totalStudentsCount}</p>
                    </div>
                    <div className={subtleMetricCardClass}>
                      <p className={statLabelClass}>Total No. of Attended Students</p>
                      <p className="mt-1 text-base font-bold text-gray-900">{attendedStudentsCount}</p>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div className="overflow-x-auto rounded-xl border border-gray-400 bg-white">
              <table className="min-w-full text-sm">
                <thead className="bg-gray-50">
                  <tr className="border-b border-gray-400">
                    <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-[0.08em] text-gray-600">
                      CO Attainment Summary
                    </th>
                    {coSummaryRows.map((item) => (
                      <th
                        key={`summary-head-${item.key}`}
                        className="px-4 py-3 text-center text-xs font-semibold uppercase tracking-[0.08em] text-gray-600"
                      >
                        {item.label}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody className="bg-white">
                  <tr className="border-b border-gray-100">
                    <td className="px-4 py-3 font-medium text-gray-800">Total No. of Students</td>
                    {coSummaryRows.map((item) => (
                      <td key={`summary-total-${item.key}`} className="px-4 py-3 text-center text-gray-900">
                        {item.totalStudents}
                      </td>
                    ))}
                  </tr>
                  <tr className="border-b border-gray-100">
                    <td className="px-4 py-3 font-medium text-gray-800">Total No. of Students Achieved the Target CO%</td>
                    {coSummaryRows.map((item) => (
                      <td key={`summary-achieved-${item.key}`} className="px-4 py-3 text-center text-gray-900">
                        {item.achievedCount}
                      </td>
                    ))}
                  </tr>
                  <tr className="border-b border-gray-100">
                    <td className="px-4 py-3 font-semibold text-gray-800">Attainment Level</td>
                    {coSummaryRows.map((item) => (
                      <td key={`summary-level-${item.key}`} className="bg-yellow-100 px-4 py-3 text-center font-bold text-yellow-800">
                        {item.attainmentLevel}
                      </td>
                    ))}
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium text-gray-800">Total No. of Students who need Remedial Class</td>
                    {coSummaryRows.map((item) => (
                      <td key={`summary-remedial-${item.key}`} className="px-4 py-3 text-center text-gray-900">
                        {item.remedialCount}
                      </td>
                    ))}
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}

      {error && (
        <div className="rounded-2xl border border-red-200 bg-red-50/90 px-5 py-4 text-red-700 shadow-[0_12px_24px_-18px_rgba(185,28,28,0.45)]">
          <p className="text-sm font-medium">{error}</p>
        </div>
      )}

      {(loadingTestTypes || loadingRows) && (
        <div className={`${sectionCardClass} p-10`}>
          <div className="flex flex-col items-center justify-center text-center">
            <svg className="animate-spin h-10 w-10 text-purple-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
            <p className="mt-4 text-sm font-medium text-gray-700">
              {loadingTestTypes ? 'Loading test types...' : 'Loading course-wise student attainment data...'}
            </p>
          </div>
        </div>
      )}

      {!loadingTestTypes && !loadingRows && !error && !testTypes.length && (
        <div className={emptyStateCardClass}>
          <p className="text-base font-semibold text-gray-800">No test types available</p>
          <p className="mt-2 text-sm text-gray-500">
            No active assessment components were found for this course.
          </p>
        </div>
      )}

      {!loadingTestTypes && !loadingRows && !error && testTypes.length > 0 && !selectedTestTypeId && (
        <div className={emptyStateCardClass}>
          <p className="text-base font-semibold text-gray-800">Select a test type</p>
          <p className="mt-2 text-sm text-gray-500">
            Choose a test type to load all students mapped to this course from every teacher allocation.
          </p>
        </div>
      )}

      {!loadingTestTypes && !loadingRows && !error && selectedTestTypeId && rows.length === 0 && (
        <div className={emptyStateCardClass}>
          <p className="text-base font-semibold text-gray-800">No present students found</p>
          <p className="mt-2 text-sm text-gray-500">
            No present students are available for this course and selected test type after excluding absentees.
          </p>
        </div>
      )}

      {!loadingTestTypes && !loadingRows && !error && rows.length > 0 && (
        <div className={`${sectionCardClass} overflow-hidden`}>
          <div className="flex flex-col gap-3 border-b border-gray-200 bg-gray-50 px-5 py-4 md:flex-row md:items-center md:justify-between">
            <div className="flex items-center gap-3">
              <h4 className="text-lg font-medium text-gray-900">Mapped Students</h4>
              <span className="inline-flex items-center rounded-full border border-purple-200 bg-purple-50 px-2.5 py-1 text-xs font-semibold text-purple-700">
                Showing {sortedFilteredRows.length} of {rows.length}
              </span>
            </div>
            <div className="w-full md:max-w-2xl">
              <div className="flex flex-col gap-2 md:flex-row md:justify-end">
                <label htmlFor="studentSearch" className="sr-only">
                  Search by Name or Register No
                </label>
                <input
                  id="studentSearch"
                  type="text"
                  value={searchQuery}
                  onChange={(event) => setSearchQuery(event.target.value)}
                  placeholder="Search by name or register number"
                  className={`${fieldClass} md:w-72`}
                />
                {showGeneratedSummary && (
                  <button
                    type="button"
                    onClick={handleDownloadAttainmentXlsx}
                    disabled={downloadingXlsx || !selectedTestTypeId || visibleColumns.length === 0 || !isPT1Selected}
                    className="inline-flex h-11 items-center justify-center gap-2 rounded-lg border border-purple-200 bg-purple-50 px-4 text-sm font-semibold text-purple-700 transition hover:bg-purple-100 disabled:cursor-not-allowed disabled:opacity-60"
                  >
                    <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 3v12m0 0l-4-4m4 4l4-4M5 21h14" />
                    </svg>
                    {downloadingXlsx ? 'Preparing...' : 'Download XLSX'}
                  </button>
                )}
              </div>
            </div>
          </div>
          {sortedFilteredRows.length === 0 ? (
            <div className="p-10 text-center">
              <p className="text-base font-semibold text-gray-800">No matching students</p>
              <p className="mt-2 text-sm text-gray-500">
                Try a different student name or register number.
              </p>
            </div>
          ) : (
            <div className="max-h-[68vh] overflow-auto">
              <table className="min-w-full text-sm">
                <thead className="sticky top-0 z-20">
                  <tr className="h-10 border-b border-gray-200 text-xs font-semibold uppercase tracking-[0.08em] text-gray-600">
                    <th
                      rowSpan={2}
                      className="h-20 bg-gray-100 px-4 py-0 text-left align-middle"
                    >
                      Register No
                    </th>
                    <th
                      rowSpan={2}
                      className="h-20 bg-gray-100 px-4 py-0 text-left align-middle"
                    >
                      Student Name
                    </th>
                    <th
                      colSpan={visibleColumns.length || 1}
                      className="h-10 bg-gray-100 px-4 py-0 text-center align-middle"
                    >
                      Marks Allocated
                    </th>
                    <th
                      colSpan={visibleColumns.length || 1}
                      className="h-10 bg-white px-4 py-0 text-center align-middle"
                    >
                      Marks Obtained
                    </th>
                    <th
                      colSpan={visibleColumns.length || 1}
                      className="h-10 bg-purple-50 px-4 py-0 text-center text-purple-800 align-middle"
                    >
                      CO Attainment %
                    </th>
                    <th
                      colSpan={visibleColumns.length || 1}
                      className="h-10 bg-purple-100 px-4 py-0 text-center text-purple-800 align-middle"
                    >
                      Attainment of COs
                    </th>
                    <th className="h-20 bg-gray-100 px-4 py-0 text-center align-middle">
                      {marksColumnLabel}
                    </th>
                  </tr>
                  <tr className="h-10 border-b border-gray-200 text-xs font-semibold uppercase tracking-[0.08em] text-gray-600">
                    {visibleColumns.map((column) => (
                      <th
                        key={`allocated-${column.key}`}
                        className="h-10 bg-gray-50 px-4 py-0 text-right align-middle"
                      >
                        {column.label}
                      </th>
                    ))}
                    {visibleColumns.map((column) => (
                      <th
                        key={`obtained-${column.key}`}
                        className="h-10 bg-white px-4 py-0 text-right align-middle"
                      >
                        {column.label}
                      </th>
                    ))}
                    {visibleColumns.map((column) => (
                      <th
                        key={`percent-${column.key}`}
                        className="h-10 bg-purple-50 px-4 py-0 text-right text-purple-800 align-middle"
                      >
                        {column.label}
                      </th>
                    ))}
                    {visibleColumns.map((column) => (
                      <th
                        key={`attainment-${column.key}`}
                        className="h-10 bg-purple-100 px-4 py-0 text-right text-purple-800 align-middle"
                      >
                        {column.label}
                      </th>
                    ))}
                    <th className="h-10 bg-gray-50 px-4 py-0 text-right align-middle">
                      Marks
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {sortedFilteredRows.map((row, index) => (
                    <tr
                      key={`${row.course_id}-${row.student_id}-${row.test_type_id}`}
                      className={`border-b border-gray-100 ${index % 2 === 0 ? 'bg-white' : 'bg-gray-50/70'} hover:bg-purple-50/60`}
                    >
                      <td className="px-4 py-3 font-medium text-gray-900">{row.register_no || '-'}</td>
                      <td className="px-4 py-3 text-gray-800">{row.student_name || '-'}</td>
                      {visibleColumns.map((column) => (
                        <td key={`${row.student_id}-allocated-${column.key}`} className="bg-gray-50/80 px-4 py-3 text-right text-gray-700">
                          {formatMarks(getColumnMaxMarks(column))}
                        </td>
                      ))}
                      {visibleColumns.map((column) => (
                        <td key={`${row.student_id}-obtained-${column.key}`} className="px-4 py-3 text-right font-semibold text-gray-900">
                          {formatMarks(getColumnObtainedMark(row, column))}
                        </td>
                      ))}
                      {visibleColumns.map((column) => {
                        const percentage = calculateAttainmentPercent(getColumnObtainedMark(row, column), getColumnMaxMarks(column))
                        const isCompared = appliedTargetPercent !== null && percentage !== null
                        const isAchieved = isCompared && percentage >= appliedTargetPercent
                        const percentageTone = isCompared
                          ? (isAchieved ? 'bg-emerald-50 text-emerald-700' : 'bg-rose-50 text-rose-700')
                          : 'bg-purple-50/70 text-gray-700'
                        return (
                          <td key={`${row.student_id}-percent-${column.key}`} className={`px-4 py-3 text-right ${percentageTone}`}>
                            {formatPercentage(percentage)}
                          </td>
                        )
                      })}
                      {visibleColumns.map((column) => {
                        const percentage = calculateAttainmentPercent(getColumnObtainedMark(row, column), getColumnMaxMarks(column))
                        const isCompared = appliedTargetPercent !== null && percentage !== null
                        const isAchieved = isCompared && percentage >= appliedTargetPercent
                        const attainmentTone = isCompared
                          ? (isAchieved ? 'bg-emerald-100 text-emerald-800' : 'bg-rose-100 text-rose-800')
                          : 'bg-gray-50 text-gray-900'
                        return (
                          <td key={`${row.student_id}-attainment-${column.key}`} className={`px-4 py-3 text-right font-semibold ${attainmentTone}`}>
                            {isCompared ? (isAchieved ? '1' : '0') : '-'}
                          </td>
                        )
                      })}
                      <td className="bg-gray-100/80 px-4 py-3 text-right font-semibold text-gray-900">
                        {formatMarks(
                          visibleColumns.reduce((sum, column) => {
                            const mark = getColumnObtainedMark(row, column)
                            return mark === null ? sum : sum + mark
                          }, 0)
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {!loadingTestTypes && !loadingRows && !error && appliedTargetPercent !== null && orderedPOSummary.length > 0 && (
        <div className={`${sectionCardClass} overflow-hidden`}>
          <div className={sectionHeaderClass}>
            <h4 className="text-base font-semibold text-gray-900">PO Attainment Summary (%)</h4>
          </div>
          <div className="grid grid-cols-1 gap-4 p-5 sm:grid-cols-2 lg:grid-cols-4">
            {orderedPOSummary.map((item) => (
              <div key={item.key || item.label} className={metricCardClass}>
                <p className={statLabelClass}>{item.label || item.key}</p>
                <p className="mt-1 text-base font-semibold text-gray-900">
                  {formatPercentage(item.attainment_percent ?? item.attainmentPercent)}
                </p>
                <div className="mt-2 h-1.5 rounded-full bg-gray-200">
                  <div
                    className="h-1.5 rounded-full bg-purple-600"
                    style={{
                      width: `${Math.max(0, Math.min(100, Number(item.attainment_percent ?? item.attainmentPercent) || 0))}%`
                    }}
                  />
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {showScrollTop && (
        <button
          type="button"
          onClick={handleScrollToTop}
          className="fixed bottom-6 right-6 z-50 inline-flex items-center justify-center rounded-full bg-purple-600 px-4 py-3 text-sm font-semibold text-white shadow-md transition-transform hover:-translate-y-0.5 hover:bg-purple-700"
          aria-label="Scroll to top"
        >
          <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 15l7-7 7 7" />
          </svg>
        </button>
      )}
    </div>
  )
}

export default COPOAttainmentSection
