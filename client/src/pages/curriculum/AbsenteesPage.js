import React, { useState, useEffect, useCallback } from 'react'
import * as XLSX from 'xlsx'
import { API_BASE_URL } from '../../config'

const LEARNING_MODES = [
  { id: 2, label: 'PBL' },
  { id: 1, label: 'UAL' },
]

export default function AbsenteesPage() {
  const userRole = (localStorage.getItem('userRole') || '').toLowerCase()

  if (!['coe', 'admin'].includes(userRole)) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-gray-50">
        <div className="text-center">
          <div className="text-5xl mb-4">🔒</div>
          <h2 className="text-xl font-semibold text-gray-700">Access Restricted</h2>
          <p className="text-gray-500 mt-1">This page is only available to COE and Admin users.</p>
        </div>
      </div>
    )
  }

  return <AbsenteesContent />
}

function AbsenteesContent() {
  /* ── selection ── */
  const [learningModeId, setLearningModeId] = useState(2)
  const [markCategories, setMarkCategories] = useState([])
  const [selectedGroupIds, setSelectedGroupIds] = useState('')
  const [selectedGroupLabel, setSelectedGroupLabel] = useState('')

  /* ── file ── */
  const [file, setFile] = useState(null)

  /* ── step: 'idle' | 'previewing' | 'confirmed' ── */
  const [step, setStep] = useState('idle')
  const [parsing, setParsing] = useState(false)
  const [previewRows, setPreviewRows] = useState([])
  const [previewErrors, setPreviewErrors] = useState([])

  /* ── confirm/upload ── */
  const [uploading, setUploading] = useState(false)
  const [uploadResult, setUploadResult] = useState(null)

  /* ── recorded absentees table ── */
  const [absentees, setAbsentees] = useState([])
  const [loadingAbsentees, setLoadingAbsentees] = useState(false)
  const [filterWindowId, setFilterWindowId] = useState('')
  const [expandedCardKey, setExpandedCardKey] = useState(null)

  /* ── fetch categories ── */
  useEffect(() => {
    setMarkCategories([])
    setSelectedGroupIds('')
    setSelectedGroupLabel('')
    setFile(null)
    setStep('idle')
    setPreviewRows([])
    setPreviewErrors([])
    setUploadResult(null)
    fetch(`${API_BASE_URL}/mark-categories/by-learning-mode?learning_mode_id=${learningModeId}`)
      .then(r => r.json())
      .then(data => setMarkCategories(Array.isArray(data) ? data : []))
      .catch(err => console.error('Error fetching categories:', err))
  }, [learningModeId])

  /* ── group categories ── */
  const getCategoryGroup = (name) => {
    const match = name.match(/^(.+?)\s*->\s*.+$/)
    return match ? match[1].trim() : name.trim()
  }
  const abbreviateGroup = (g) => g.replace(/\bPeriodical Test\b/gi, 'PT -').trim()

  const categoryGroupOptions = markCategories.reduce((groups, cat) => {
    const groupName = getCategoryGroup(cat.name)
    const existing = groups.find(g => g.groupName === groupName)
    if (existing) {
      if (!existing.ids.includes(cat.id)) existing.ids.push(cat.id) // dedup
    } else {
      groups.push({ groupName, label: abbreviateGroup(groupName), ids: [cat.id] })
    }
    return groups
  }, [])

  /* ── fetch recorded absentees ── */
  const fetchAbsentees = useCallback(() => {
    setLoadingAbsentees(true)
    const params = new URLSearchParams({ learning_mode_id: learningModeId })
    if (filterWindowId) params.append('window_id', filterWindowId)
    fetch(`${API_BASE_URL}/exam-absentees?${params}`)
      .then(r => r.json())
      .then(data => {
        if (!Array.isArray(data)) { setAbsentees([]); return }
        const seen = new Set()
        setAbsentees(data.filter(a => {
          const k = `${a.window_id}_${a.course_id}_${a.student_id}_${a.mark_category_id}`
          if (seen.has(k)) return false
          seen.add(k)
          return true
        }))
      })
      .catch(err => console.error('Error fetching absentees:', err))
      .finally(() => setLoadingAbsentees(false))
  }, [learningModeId, filterWindowId])

  useEffect(() => { fetchAbsentees() }, [fetchAbsentees])

  /* ── reset when file or group changes ── */
  const resetUploadState = () => {
    setStep('idle')
    setPreviewRows([])
    setPreviewErrors([])
    setUploadResult(null)
  }

  /* ── step 1: parse & preview ── */
  const handleParsePreview = async () => {
    if (!file || !selectedGroupIds) return
    setParsing(true)
    resetUploadState()

    const form = new FormData()
    form.append('file', file)
    form.append('mark_category_ids', selectedGroupIds)
    form.append('learning_mode_id', learningModeId)

    try {
      const res = await fetch(`${API_BASE_URL}/exam-absentees/preview`, { method: 'POST', body: form })
      if (!res.ok) {
        const txt = await res.text()
        setPreviewErrors([{ row: 0, message: txt }])
        setStep('previewing')
        return
      }
      const data = await res.json()
      setPreviewRows(data.rows || [])
      setPreviewErrors(data.error_rows || [])
      setStep('previewing')
    } catch (err) {
      setPreviewErrors([{ row: 0, message: err.message }])
      setStep('previewing')
    } finally {
      setParsing(false)
    }
  }

  /* ── step 2: confirm & save ── */
  const handleConfirmSave = async () => {
    if (!file || !selectedGroupIds) return
    setUploading(true)
    setUploadResult(null)

    const form = new FormData()
    form.append('file', file)
    form.append('mark_category_ids', selectedGroupIds)
    form.append('learning_mode_id', learningModeId)

    try {
      const res = await fetch(`${API_BASE_URL}/exam-absentees/upload`, { method: 'POST', body: form })
      if (!res.ok) {
        const txt = await res.text()
        setUploadResult({ success: false, error: txt })
      } else {
        const data = await res.json()
        setUploadResult(data)
        if (data.inserted > 0) fetchAbsentees()
        setStep('confirmed')
      }
    } catch (err) {
      setUploadResult({ success: false, error: err.message })
    } finally {
      setUploading(false)
    }
  }

  /* ── delete ── */
  const handleDelete = async (id) => {
    if (!window.confirm('Remove this absentee record?')) return
    try {
      await fetch(`${API_BASE_URL}/exam-absentees/${id}`, { method: 'DELETE' })
      setAbsentees(prev => prev.filter(a => a.id !== id))
    } catch (err) { console.error('Delete error:', err) }
  }

  /* ── bulk delete by window ── */
  const handleDeleteByWindow = async (windowId) => {
    if (!window.confirm(`Remove ALL absentee records for Exam ${windowId}? This cannot be undone.`)) return
    try {
      const res = await fetch(`${API_BASE_URL}/exam-absentees/by-window/${windowId}`, { method: 'DELETE' })
      if (res.ok) {
        setAbsentees(prev => prev.filter(a => a.window_id !== windowId))
      }
    } catch (err) { console.error('Bulk delete error:', err) }
  }

  const selectedMode = LEARNING_MODES.find(m => m.id === learningModeId)
  const canPreview = file && selectedGroupIds && step === 'idle'
  const canConfirm = step === 'previewing' && previewRows.length > 0

  const handleDownloadTemplate = () => {
    const ws = XLSX.utils.aoa_to_sheet([
      ['exam_id', 'course_id', 'student_id'],
    ])
    // Set column widths for readability
    ws['!cols'] = [{ wch: 14 }, { wch: 14 }, { wch: 16 }]
    const wb = XLSX.utils.book_new()
    XLSX.utils.book_append_sheet(wb, ws, 'Absentees Template')
    XLSX.writeFile(wb, 'exam_absentees_report.xlsx')
  }

  return (
    <div className="min-h-screen bg-gray-50 p-6">
      {/* ── header ── */}
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-gray-900">Exam Absentees</h1>
        <p className="text-gray-500 mt-1 text-sm">
          Upload an absentee sheet to block mark entry for specific students in a given exam window.
        </p>
      </div>

      {/* ── Upload card ── */}
      <div className="bg-white rounded-2xl shadow-sm border border-gray-200 p-6 mb-8">
        <div className="flex items-center justify-between mb-5">
          <h2 className="text-base font-semibold text-gray-800">Upload Absentees</h2>
          <button
            onClick={handleDownloadTemplate}
            className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-violet-700 bg-violet-50 hover:bg-violet-100 border border-violet-200 rounded-lg transition-colors"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
            </svg>
            Download Template
          </button>
        </div>

        {/* Controls row */}
        <div className="flex flex-wrap items-end gap-6">
          {/* Learning mode toggle */}
          <div>
            <label className="block text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">
              Learning Mode
            </label>
            <div className="flex items-center bg-gray-100 rounded-xl p-1 gap-1">
              {LEARNING_MODES.map(mode => (
                <button
                  key={mode.id}
                  onClick={() => { setLearningModeId(mode.id); resetUploadState() }}
                  className={`px-5 py-2 rounded-lg text-sm font-semibold transition-all ${
                    learningModeId === mode.id
                      ? 'bg-violet-600 text-white shadow-sm'
                      : 'text-gray-500 hover:text-gray-700'
                  }`}
                >
                  {mode.label}
                </button>
              ))}
            </div>
          </div>

          {/* Component dropdown */}
          <div className="flex-1 min-w-[220px] max-w-sm">
            <label className="block text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">
              Mark Component
            </label>
            <select
              value={selectedGroupIds}
              onChange={e => {
                setSelectedGroupIds(e.target.value)
                const opt = categoryGroupOptions.find(g => g.ids.join(',') === e.target.value)
                setSelectedGroupLabel(opt ? opt.label : '')
                resetUploadState()
              }}
              className="w-full border border-gray-300 rounded-lg px-3 py-2.5 text-sm text-gray-700 focus:outline-none focus:ring-2 focus:ring-violet-500 bg-white"
            >
              <option value="">— Select component —</option>
              {categoryGroupOptions.map(group => (
                <option key={group.groupName} value={group.ids.join(',')}>
                  {group.label}
                </option>
              ))}
            </select>
          </div>

          {/* File picker */}
          {selectedGroupIds && (
            <div>
              <label className="block text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">
                Absentee Sheet (.xlsx)
              </label>
              <label className="flex items-center gap-2 cursor-pointer">
                <span className="px-4 py-2.5 bg-gray-100 hover:bg-gray-200 text-gray-700 text-sm font-medium rounded-lg border border-gray-300 transition-colors">
                  {file ? '📄 ' + file.name : '📎 Choose File'}
                </span>
                <input
                  type="file"
                  accept=".xlsx,.xls"
                  className="hidden"
                  onChange={e => { setFile(e.target.files[0] || null); resetUploadState() }}
                />
              </label>
            </div>
          )}

          {/* Parse & Preview button */}
          {canPreview && (
            <button
              onClick={handleParsePreview}
              disabled={parsing}
              className="px-6 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 text-white text-sm font-semibold rounded-lg shadow-sm transition-colors"
            >
              {parsing ? 'Parsing…' : '🔍 Parse & Preview'}
            </button>
          )}

          {/* Confirm & Save button */}
          {canConfirm && !uploading && (
            <button
              onClick={handleConfirmSave}
              className="px-6 py-2.5 bg-violet-600 hover:bg-violet-700 text-white text-sm font-semibold rounded-lg shadow-sm transition-colors"
            >
              ✅ Confirm & Save
            </button>
          )}
          {uploading && (
            <span className="px-6 py-2.5 bg-violet-400 text-white text-sm font-semibold rounded-lg">
              Saving…
            </span>
          )}
        </div>

        {/* ── Preview table ── */}
        {step === 'previewing' && (
          <div className="mt-6">
            <div className="flex items-center gap-3 mb-3">
              <h3 className="text-sm font-semibold text-gray-700">
                Preview — {previewRows.length} record{previewRows.length !== 1 ? 's' : ''} will be saved
              </h3>
              {previewErrors.length > 0 && (
                <span className="px-2 py-0.5 bg-red-100 text-red-700 text-xs font-medium rounded-full">
                  {previewErrors.length} row error{previewErrors.length !== 1 ? 's' : ''}
                </span>
              )}
              <button
                onClick={resetUploadState}
                className="ml-auto text-xs text-blue-600 hover:underline"
              >
                ← Re-parse with different file
              </button>
            </div>

            {previewErrors.length > 0 && (
              <div className="mb-3 p-3 bg-red-50 border border-red-200 rounded-lg text-xs text-red-700">
                <p className="font-semibold mb-1">Rows that could not be resolved:</p>
                <ul className="list-disc list-inside space-y-0.5">
                  {previewErrors.map((e, i) => (
                    <li key={i}>{e.row > 0 ? `Row ${e.row}: ` : ''}{e.message}</li>
                  ))}
                </ul>
              </div>
            )}

            {previewRows.length === 0 && previewErrors.length > 0 && (
              <div className="p-4 bg-red-50 border border-red-200 rounded-xl text-sm text-red-700 font-medium">
                ⚠️ All rows had errors — nothing will be saved. Fix the file and re-parse.
              </div>
            )}

            {previewRows.length > 0 && (
              <div className="overflow-x-auto rounded-xl border border-gray-200">
                <table className="w-full text-sm divide-y divide-gray-200">
                  <thead className="bg-blue-50">
                    <tr>
                      {['#', 'Exam ID', 'Course', 'Student', 'Register No', 'Component'].map(h => (
                        <th key={h} className="px-4 py-2.5 text-left text-xs font-semibold text-blue-700 uppercase tracking-wider">
                          {h}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody className="bg-white divide-y divide-gray-100">
                    {(() => {
                      // Group preview rows by student + course + PT type
                      const groupMap = new Map()

                      previewRows.forEach(row => {
                        const nameUpper = (row.category_name || '').toUpperCase()
                        
                        // Detect PT-2 first to avoid false matches with PT-1
                        const isPT2 = nameUpper.includes('PERIODICAL TEST 2') ||
                                      nameUpper.includes('PERIODICALTEST2') ||
                                      nameUpper.includes('TEST 2') ||
                                      (nameUpper.includes('PT') && nameUpper.match(/PT[\s-]*2/))
                        
                        // Detect PT-1 variants (exclude PT-2)
                        const isPT1 = !isPT2 && (
                                      nameUpper.includes('PERIODICAL TEST 1') ||
                                      nameUpper.includes('PERIODICALTEST1') ||
                                      nameUpper.includes('TEST 1') ||
                                      (nameUpper.includes('PT') && nameUpper.match(/PT[\s-]*1/))
                                    )
                        
                        let groupKey
                        let displayName
                        
                        if (isPT1) {
                          groupKey = `${row.register_no}_${row.course_code}_PT1`
                          displayName = 'PT - 1'
                        } else if (isPT2) {
                          groupKey = `${row.register_no}_${row.course_code}_PT2`
                          displayName = 'PT - 2'
                        } else {
                          // Non-PT components: keep separate
                          groupKey = `${row.register_no}_${row.course_code}_${row.category_name}`
                          displayName = row.category_name
                        }

                        if (!groupMap.has(groupKey)) {
                          groupMap.set(groupKey, {
                            ...row,
                            displayName,
                          })
                        }
                      })

                      const groupedRows = Array.from(groupMap.values())

                      return groupedRows.map((row, idx) => (
                        <tr key={idx} className={idx % 2 === 0 ? 'bg-white' : 'bg-gray-50'}>
                          <td className="px-4 py-2.5 text-gray-400 text-xs">{idx + 1}</td>
                          <td className="px-4 py-2.5 font-mono text-gray-700">{row.exam_id}</td>
                          <td className="px-4 py-2.5">
                            <div className="font-medium text-gray-800">{row.course_code}</div>
                            <div className="text-xs text-gray-400">{row.course_name}</div>
                          </td>
                          <td className="px-4 py-2.5 text-gray-700">{row.student_name}</td>
                          <td className="px-4 py-2.5 font-mono text-gray-600 text-xs">{row.register_no}</td>
                          <td className="px-4 py-2.5">
                            <span className="px-2 py-0.5 bg-violet-100 text-violet-700 text-xs font-medium rounded-full">
                              {row.displayName}
                            </span>
                          </td>
                        </tr>
                      ))
                    })()}
                  </tbody>
                </table>
              </div>
            )}

            {previewRows.length > 0 && (
              <p className="mt-3 text-sm text-gray-500">
                Review the {previewRows.length} record{previewRows.length !== 1 ? 's' : ''} above, then click{' '}
                <strong className="text-violet-700">Confirm & Save</strong> to lock them in.
              </p>
            )}
          </div>
        )}

        {/* Upload result */}
        {uploadResult && (
          <div className={`mt-5 p-4 rounded-xl border text-sm ${
            uploadResult.success
              ? 'bg-green-50 border-green-200 text-green-800'
              : 'bg-red-50 border-red-200 text-red-800'
          }`}>
            {uploadResult.error ? (
              <p><strong>Error:</strong> {uploadResult.error}</p>
            ) : (
              <>
                <p className="font-semibold mb-1">
                  {uploadResult.success ? '✅ Saved successfully' : '⚠️ Saved with errors'}
                </p>
                <p>
                  Total XL rows: <strong>{uploadResult.total_rows}</strong>&nbsp;|&nbsp;
                  Inserted: <strong className="text-green-700">{uploadResult.inserted}</strong>&nbsp;|&nbsp;
                  Skipped (duplicate): <strong>{uploadResult.skipped}</strong>
                </p>
                {uploadResult.error_rows?.length > 0 && (
                  <ul className="mt-2 list-disc list-inside space-y-0.5">
                    {uploadResult.error_rows.map((e, i) => (
                      <li key={i}>Row {e.row}: {e.message}</li>
                    ))}
                  </ul>
                )}
              </>
            )}
          </div>
        )}
      </div>

      {/* ── Recorded Absentees cards ── */}
      <RecordedAbsenteesSection
        absentees={absentees}
        loadingAbsentees={loadingAbsentees}
        filterWindowId={filterWindowId}
        setFilterWindowId={setFilterWindowId}
        fetchAbsentees={fetchAbsentees}
        expandedCardKey={expandedCardKey}
        setExpandedCardKey={setExpandedCardKey}
        handleDelete={handleDelete}
        handleDeleteByWindow={handleDeleteByWindow}
      />
    </div>
  )
}

/* ─────────────────────────────────────────────────────────────────────────── */
/*  Recorded Absentees — card grid + expandable detail                        */
/* ─────────────────────────────────────────────────────────────────────────── */
function RecordedAbsenteesSection({
  absentees, loadingAbsentees,
  filterWindowId, setFilterWindowId,
  fetchAbsentees,
  expandedCardKey, setExpandedCardKey,
  handleDelete,
  handleDeleteByWindow,
}) {
  /* Group by window_id only — one card per exam */
  const groups = React.useMemo(() => {
    const map = new Map()
    absentees.forEach(a => {
      const key = `${a.window_id}`
      if (!map.has(key)) {
        map.set(key, {
          key,
          window_id: a.window_id,
          window_name: a.window_name || '',
          earliest: a.created_at,
          window_end_at: a.window_end_at || null,
          rows: [],
        })
      }
      const g = map.get(key)
      if (a.window_name && !g.window_name) g.window_name = a.window_name
      g.rows.push(a)
      if (new Date(a.created_at) < new Date(g.earliest)) g.earliest = a.created_at
      // Keep the latest window_end_at across rows
      if (a.window_end_at && (!g.window_end_at || new Date(a.window_end_at) > new Date(g.window_end_at))) {
        g.window_end_at = a.window_end_at
      }
    })
    return Array.from(map.values()).sort((a, b) => b.window_id - a.window_id)
  }, [absentees])

  const toggleCard = (key) =>
    setExpandedCardKey(prev => (prev === key ? null : key))

  return (
    <div className="bg-white rounded-2xl shadow-sm border border-gray-200 p-6">
      {/* header */}
      <div className="flex items-center justify-between mb-5 flex-wrap gap-3">
        <h2 className="text-base font-semibold text-gray-800">
          Recorded Absentees
          {groups.length > 0 && (
            <span className="ml-2 px-2 py-0.5 text-xs bg-violet-100 text-violet-700 rounded-full font-medium">
              {groups.length} exam{groups.length !== 1 ? 's' : ''}
            </span>
          )}
        </h2>
        <div className="flex items-center gap-3">
          <input
            type="number"
            placeholder="Filter by Exam ID"
            value={filterWindowId}
            onChange={e => setFilterWindowId(e.target.value)}
            className="w-44 border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-violet-500"
          />
          <button
            onClick={fetchAbsentees}
            className="px-4 py-2 bg-gray-100 hover:bg-gray-200 text-gray-700 text-sm font-medium rounded-lg transition-colors"
          >
            Refresh
          </button>
        </div>
      </div>

      {loadingAbsentees ? (
        <div className="py-16 text-center text-gray-400">Loading…</div>
      ) : groups.length === 0 ? (
        <div className="py-16 text-center">
          <div className="text-4xl mb-3">📋</div>
          <p className="text-gray-500 text-sm">No absentees recorded yet for the current filters.</p>
        </div>
      ) : (
        <div className="space-y-4">
          {groups.map(group => {
            const isOpen = expandedCardKey === group.key

            /* unique courses in this exam */
            const courseSet = [...new Map(
              group.rows.map(r => [r.course_id, { code: r.course_code, name: r.course_name }])
            ).values()]

            /* unique components with PT-1/PT-2 grouping */
            const allComponents = group.rows.map(r => r.category_name).filter(Boolean)
            const componentSet = (() => {
              const grouped = []
              let seenPT1 = false
              let seenPT2 = false

              allComponents.forEach(name => {
                const nameUpper = name.toUpperCase()
                
                // Detect PT-1 variants (must check PT-2 first to avoid false matches)
                const isPT2 = nameUpper.includes('PERIODICAL TEST 2') ||
                              nameUpper.includes('PERIODICALTEST2') ||
                              nameUpper.includes('TEST 2') ||
                              (nameUpper.includes('PT') && nameUpper.match(/PT[\s-]*2/))
                
                const isPT1 = !isPT2 && (
                              nameUpper.includes('PERIODICAL TEST 1') ||
                              nameUpper.includes('PERIODICALTEST1') ||
                              nameUpper.includes('TEST 1') ||
                              (nameUpper.includes('PT') && nameUpper.match(/PT[\s-]*1/))
                            )
                
                if (isPT1 && !seenPT1) {
                  grouped.push('PT - 1')
                  seenPT1 = true
                } else if (isPT2 && !seenPT2) {
                  grouped.push('PT - 2')
                  seenPT2 = true
                } else if (!isPT1 && !isPT2) {
                  // Only add non-PT components if not already included
                  if (!grouped.includes(name)) {
                    grouped.push(name)
                  }
                }
              })

              return grouped
            })()

            /* unique modes */
            const modeSet = [...new Set(group.rows.map(r => r.learning_mode_id))]

            /* check if exam window is over */
            const isExamOver = group.window_end_at && new Date(group.window_end_at) < new Date()

            return (
              <div
                key={group.key}
                className={`rounded-xl border transition-all ${
                  isExamOver
                    ? 'border-gray-300 bg-gray-50/50'
                    : isOpen
                      ? 'border-violet-300 shadow-md'
                      : 'border-gray-200 hover:border-violet-200 hover:shadow-sm'
                }`}
              >
                {/* ── Card header (always visible, click to toggle) ── */}
                <button
                  onClick={() => toggleCard(group.key)}
                  className="w-full text-left px-5 py-4 flex items-center gap-4 rounded-xl focus:outline-none"
                >
                  {/* Exam ID badge */}
                  <div className={`flex-shrink-0 w-14 h-14 rounded-xl border flex flex-col items-center justify-center ${
                    isExamOver ? 'bg-gray-100 border-gray-300' : 'bg-violet-50 border-violet-200'
                  }`}>
                    <span className={`text-[10px] font-semibold uppercase tracking-wider leading-none ${
                      isExamOver ? 'text-gray-400' : 'text-violet-400'
                    }`}>Exam</span>
                    <span className={`text-lg font-bold leading-tight ${
                      isExamOver ? 'text-gray-500' : 'text-violet-700'
                    }`}>{group.window_id}</span>
                  </div>

                  {/* Main info */}
                  <div className="flex-1 min-w-0">
                    {group.window_name && (
                      <p className="text-sm font-semibold text-gray-800 mb-1 truncate">{group.window_name}</p>
                    )}
                    <div className="flex items-center gap-2 flex-wrap">
                      {isExamOver && (
                        <span className="px-2.5 py-0.5 bg-gray-200 text-gray-600 text-xs font-bold rounded-full uppercase tracking-wide">
                          Exam Over
                        </span>
                      )}
                      {componentSet.map(c => (
                        <span key={c} className="px-2 py-0.5 bg-violet-100 text-violet-700 text-xs font-medium rounded-full">
                          {c}
                        </span>
                      ))}
                      {modeSet.map(m => (
                        <span key={m} className={`px-2 py-0.5 text-xs font-medium rounded-full ${
                          m === 2 ? 'bg-blue-100 text-blue-700' : 'bg-emerald-100 text-emerald-700'
                        }`}>
                          {m === 2 ? 'PBL' : 'UAL'}
                        </span>
                      ))}
                    </div>
                    <div className="mt-1 text-xs text-gray-400">
                      {courseSet.length} course{courseSet.length !== 1 ? 's' : ''} &middot;&nbsp;
                      {group.rows.length} student record{group.rows.length !== 1 ? 's' : ''} &middot;&nbsp;
                      Recorded {new Date(group.earliest).toLocaleDateString(undefined, { day:'numeric', month:'short', year:'numeric' })}
                      {isExamOver && group.window_end_at && (
                        <span> &middot;&nbsp;Ended {new Date(group.window_end_at).toLocaleDateString(undefined, { day:'numeric', month:'short', year:'numeric' })}</span>
                      )}
                    </div>
                  </div>

                  {/* Remove All button */}
                  <button
                    onClick={(e) => { e.stopPropagation(); handleDeleteByWindow(group.window_id) }}
                    className="flex-shrink-0 px-3 py-1.5 text-xs text-red-500 hover:text-red-700 hover:bg-red-50 rounded-lg font-medium transition-colors"
                    title={`Remove all ${group.rows.length} records for Exam ${group.window_id}`}
                  >
                    Remove All
                  </button>

                  {/* Chevron */}
                  <svg
                    className={`w-5 h-5 text-gray-400 flex-shrink-0 transition-transform ${isOpen ? 'rotate-180' : ''}`}
                    fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}
                  >
                    <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
                  </svg>
                </button>

                {/* ── Expanded detail ── */}
                {isOpen && (() => {
                  // Group rows by student + course + PT type
                  const groupedRows = []
                  const groupMap = new Map()

                  group.rows.forEach(row => {
                    const nameUpper = (row.category_name || '').toUpperCase()
                    
                    // Detect PT-2 first to avoid false matches with PT-1
                    const isPT2 = nameUpper.includes('PERIODICAL TEST 2') ||
                                  nameUpper.includes('PERIODICALTEST2') ||
                                  nameUpper.includes('TEST 2') ||
                                  (nameUpper.includes('PT') && nameUpper.match(/PT[\s-]*2/))
                    
                    // Detect PT-1 variants (exclude PT-2)
                    const isPT1 = !isPT2 && (
                                  nameUpper.includes('PERIODICAL TEST 1') ||
                                  nameUpper.includes('PERIODICALTEST1') ||
                                  nameUpper.includes('TEST 1') ||
                                  (nameUpper.includes('PT') && nameUpper.match(/PT[\s-]*1/))
                                )
                    
                    let groupKey
                    let displayName
                    
                    if (isPT1) {
                      groupKey = `${row.student_id}_${row.course_id}_PT1`
                      displayName = 'PT - 1'
                    } else if (isPT2) {
                      groupKey = `${row.student_id}_${row.course_id}_PT2`
                      displayName = 'PT - 2'
                    } else {
                      // Non-PT components: keep separate
                      groupKey = `${row.student_id}_${row.course_id}_${row.mark_category_id}`
                      displayName = row.category_name
                    }

                    if (!groupMap.has(groupKey)) {
                      groupMap.set(groupKey, {
                        ...row,
                        displayName,
                        originalIds: [row.id],
                      })
                    } else {
                      // Add this ID to the group for batch deletion
                      groupMap.get(groupKey).originalIds.push(row.id)
                    }
                  })

                  groupedRows.push(...groupMap.values())

                  return (
                    <div className="border-t border-gray-100 px-5 pb-5 pt-4">
                      <div className="overflow-x-auto rounded-xl border border-gray-200">
                        <table className="w-full text-sm divide-y divide-gray-200">
                          <thead className="bg-violet-50">
                            <tr>
                              {['#', 'Course', 'Student', 'Register No', 'Component', 'Mode', 'Recorded At', ''].map(h => (
                                <th key={h} className="px-4 py-2.5 text-left text-xs font-semibold text-violet-700 uppercase tracking-wider">
                                  {h}
                                </th>
                              ))}
                            </tr>
                          </thead>
                          <tbody className="bg-white divide-y divide-gray-100">
                            {groupedRows.map((a, idx) => (
                              <tr key={a.originalIds.join('_')} className={idx % 2 === 0 ? 'bg-white' : 'bg-gray-50'}>
                                <td className="px-4 py-2.5 text-gray-400 text-xs">{idx + 1}</td>
                                <td className="px-4 py-2.5">
                                  <div className="font-medium text-gray-800">{a.course_code}</div>
                                  <div className="text-xs text-gray-400">{a.course_name}</div>
                                </td>
                                <td className="px-4 py-2.5 text-gray-700">{a.student_name || '—'}</td>
                                <td className="px-4 py-2.5 font-mono text-gray-600 text-xs">{a.register_no}</td>
                                <td className="px-4 py-2.5">
                                  <span className="px-2 py-0.5 bg-violet-100 text-violet-700 text-xs font-medium rounded-full">
                                    {a.displayName}
                                  </span>
                                </td>
                                <td className="px-4 py-2.5">
                                  <span className={`px-2 py-0.5 text-xs font-medium rounded-full ${
                                    a.learning_mode_id === 2 ? 'bg-blue-100 text-blue-700' : 'bg-emerald-100 text-emerald-700'
                                  }`}>
                                    {a.learning_mode_id === 2 ? 'PBL' : 'UAL'}
                                  </span>
                                </td>
                                <td className="px-4 py-2.5 text-gray-400 text-xs">
                                  {new Date(a.created_at).toLocaleString()}
                                </td>
                                <td className="px-4 py-2.5">
                                  <button
                                    onClick={() => {
                                      // Delete all records in this group
                                      a.originalIds.forEach(id => handleDelete(id))
                                    }}
                                    className="text-red-500 hover:text-red-700 text-xs font-medium hover:underline transition-colors"
                                    title={a.originalIds.length > 1 ? `Remove ${a.originalIds.length} grouped records` : 'Remove'}
                                  >
                                    Remove
                                  </button>
                                </td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    </div>
                  )
                })()}
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
