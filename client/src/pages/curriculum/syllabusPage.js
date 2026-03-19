import React, { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import MainLayout from '../../components/MainLayout'
import { API_BASE_URL } from '../../config'

function SyllabusPage() {
  const { courseId } = useParams()
  const navigate = useNavigate()
  
  // Course info
  const [courseInfo, setCourseInfo] = useState(null)
  const [curriculumTemplate, setCurriculumTemplate] = useState('2026')
  
  // Header fields (outcomes, etc.)
  const [header, setHeader] = useState({
    objectives: [],
    outcomes: [],
    reference_list: [],
    textbook_reference_list: [],
    prerequisites: [],
    teamwork: { activities: [], hours: 0 },
    selflearning: { main_inputs: [], hours: 0 }
  })
  
  // Models with nested structure: models → titles → topics
  const [models, setModels] = useState([])
  
  // Experiments for 2022 template
  const [experiments, setExperiments] = useState([])
  
  // UI state
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [activeTab, setActiveTab] = useState('objectives') // Will be updated based on template
  const [expandedModels, setExpandedModels] = useState({})
  const [expandedTitles, setExpandedTitles] = useState({})
  const [expandedExperiments, setExpandedExperiments] = useState({})

  useEffect(() => {
    fetchCourseInfo()
    fetchSyllabus()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [courseId])

  // Update active tab when curriculum template changes
  useEffect(() => {
    // Set default tab based on curriculum template
    if (curriculumTemplate === '2022') {
      setActiveTab('objectives')
    } else {
      setActiveTab('outcomes')
    }
  }, [curriculumTemplate])

  // Fetch course information
  const fetchCourseInfo = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/course/${courseId}`)
      if (!response.ok) throw new Error('Failed to fetch course info')
      const data = await response.json()
      setCourseInfo(data)
    } catch (err) {
      console.error('Error fetching course info:', err)
    }
  }

  // Fetch complete nested syllabus structure
  const fetchSyllabus = async () => {
    try {
      setLoading(true)
      const response = await fetch(`${API_BASE_URL}/course/${courseId}/syllabus`)
      if (!response.ok) throw new Error('Failed to fetch syllabus')
      
      const data = await response.json()
      
      setHeader({
        objectives: data.header?.objectives || [],
        outcomes: data.header?.outcomes || [],
        reference_list: data.header?.reference_list || [],
        textbook_reference_list: data.header?.textbook_reference_list || [],
        prerequisites: data.header?.prerequisites || [],
        teamwork: data.header?.teamwork || { activities: [], hours: 0 },
        selflearning: data.header?.selflearning || { main_inputs: [], hours: 0 }
      })
      
      setModels(data.models || [])
      setExperiments(data.experiments || [])
      
      if (data.curriculum_template) {
        setCurriculumTemplate(data.curriculum_template)
      }
      
      setError('')
    } catch (err) {
      console.error('Error fetching syllabus:', err)
      setError('Failed to load syllabus')
    } finally {
      setLoading(false)
    }
  }

  // Save header fields (outcomes, etc.)
  const handleSaveHeader = async (e) => {
    e.preventDefault()
    
    try {
      const response = await fetch(`${API_BASE_URL}/course/${courseId}/syllabus`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(header),
      })

      if (!response.ok) throw new Error('Failed to save syllabus')

      setSuccess('Syllabus header saved successfully!')
      setTimeout(() => setSuccess(''), 3000)
      setError('')
    } catch (err) {
      console.error('Error saving syllabus:', err)
      setError('Failed to save syllabus header')
    }
  }

  // Header field management
  const addHeaderItem = (field) => {
    setHeader({ ...header, [field]: [...header[field], ''] })
  }

  const removeHeaderItem = (field, index) => {
    setHeader({ ...header, [field]: header[field].filter((_, i) => i !== index) })
  }

  const updateHeaderItem = (field, index, value) => {
    const updated = [...header[field]]
    updated[index] = value
    setHeader({ ...header, [field]: updated })
  }

  // ============================================================================
  // TEAM WORK OPERATIONS
  // ============================================================================

  const addTeamworkActivity = () => {
    setHeader({ 
      ...header, 
      teamwork: {
        ...header.teamwork,
        activities: [...header.teamwork.activities, '']
      }
    })
  }

  const removeTeamworkActivity = (index) => {
    setHeader({ 
      ...header, 
      teamwork: {
        ...header.teamwork,
        activities: header.teamwork.activities.filter((_, i) => i !== index)
      }
    })
  }

  const updateTeamworkActivity = (index, value) => {
    const updated = [...header.teamwork.activities]
    updated[index] = value
    setHeader({ 
      ...header, 
      teamwork: {
        ...header.teamwork,
        activities: updated
      }
    })
  }

  const updateTeamworkHours = (hours) => {
    setHeader({ 
      ...header, 
      teamwork: {
        ...header.teamwork,
        hours: parseInt(hours) || 0
      }
    })
  }

  const renderTeamwork = () => {
    return (
      <div className="space-y-4">
        {/* Header with Total Hours */}
        <div className="border border-gray-200 rounded-lg p-4 bg-blue-50 mb-6">
          <div className="flex justify-between items-center">
            <h3 className="text-lg font-semibold text-gray-900">Team Work Activities</h3>
            <div className="flex items-center space-x-3">
              <label className="text-sm font-medium text-gray-700">
                Total Hours:
              </label>
              <input
                type="number"
                min="0"
                value={header.teamwork.hours}
                onChange={(e) => updateTeamworkHours(e.target.value)}
                className="input-custom w-24"
                placeholder="0"
              />
            </div>
          </div>
        </div>

        {/* Activities List */}
        <div className="flex justify-between items-center mb-3">
          <h4 className="text-md font-medium text-gray-700">Activities</h4>
          <button
            type="button"
            onClick={addTeamworkActivity}
            className="btn-primary-custom"
          >
            + Add Activity
          </button>
        </div>

        {header.teamwork.activities.length === 0 ? (
          <p className="text-gray-500 text-center py-8">No team work activities added yet.</p>
        ) : (
          <div className="space-y-3">
            {header.teamwork.activities.map((activity, index) => (
              <div key={index} className="border border-gray-200 rounded-lg p-4 bg-gray-50">
                <div className="flex items-center space-x-4">
                  <div className="flex-1">
                    <input
                      type="text"
                      value={activity}
                      onChange={(e) => updateTeamworkActivity(index, e.target.value)}
                      className="input-custom"
                      placeholder="Enter activity name"
                    />
                  </div>
                  <button
                    type="button"
                    onClick={() => removeTeamworkActivity(index)}
                    className="text-red-600 hover:text-red-800 p-2"
                    title="Remove activity"
                  >
                    <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z" clipRule="evenodd" />
                    </svg>
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    )
  }

  // ============================================================================
  // SELF LEARNING OPERATIONS
  // ============================================================================

  const addMainInput = () => {
    setHeader({ 
      ...header, 
      selflearning: {
        ...header.selflearning,
        main_inputs: [...header.selflearning.main_inputs, { main: '', internal: [] }]
      }
    })
  }

  const removeMainInput = (index) => {
    setHeader({ 
      ...header, 
      selflearning: {
        ...header.selflearning,
        main_inputs: header.selflearning.main_inputs.filter((_, i) => i !== index)
      }
    })
  }

  const updateMainInput = (index, value) => {
    const updated = [...header.selflearning.main_inputs]
    updated[index] = { ...updated[index], main: value }
    setHeader({ 
      ...header, 
      selflearning: {
        ...header.selflearning,
        main_inputs: updated
      }
    })
  }

  const addInternalInput = (mainIndex) => {
    const updated = [...header.selflearning.main_inputs]
    updated[mainIndex] = {
      ...updated[mainIndex],
      internal: [...updated[mainIndex].internal, '']
    }
    setHeader({ 
      ...header, 
      selflearning: {
        ...header.selflearning,
        main_inputs: updated
      }
    })
  }

  const removeInternalInput = (mainIndex, internalIndex) => {
    const updated = [...header.selflearning.main_inputs]
    updated[mainIndex] = {
      ...updated[mainIndex],
      internal: updated[mainIndex].internal.filter((_, i) => i !== internalIndex)
    }
    setHeader({ 
      ...header, 
      selflearning: {
        ...header.selflearning,
        main_inputs: updated
      }
    })
  }

  const updateInternalInput = (mainIndex, internalIndex, value) => {
    const updated = [...header.selflearning.main_inputs]
    updated[mainIndex].internal[internalIndex] = value
    setHeader({ 
      ...header, 
      selflearning: {
        ...header.selflearning,
        main_inputs: updated
      }
    })
  }

  const updateSelfLearningHours = (hours) => {
    setHeader({ 
      ...header, 
      selflearning: {
        ...header.selflearning,
        hours: parseInt(hours) || 0
      }
    })
  }

  const renderSelfLearning = () => {
    return (
      <div className="space-y-4">
        {/* Header with Total Hours */}
        <div className="border border-gray-200 rounded-lg p-4 bg-green-50 mb-6">
          <div className="flex justify-between items-center">
            <h3 className="text-lg font-semibold text-gray-900">Self Learning</h3>
            <div className="flex items-center space-x-3">
              <label className="text-sm font-medium text-gray-700">
                Total Hours:
              </label>
              <input
                type="number"
                min="0"
                value={header.selflearning.hours}
                onChange={(e) => updateSelfLearningHours(e.target.value)}
                className="input-custom w-24"
                placeholder="0"
              />
            </div>
          </div>
        </div>

        {/* Main Inputs List */}
        <div className="flex justify-between items-center mb-3">
          <h4 className="text-md font-medium text-gray-700">Main Topics</h4>
          <button
            type="button"
            onClick={addMainInput}
            className="btn-primary-custom"
          >
            + Add Main Topic
          </button>
        </div>

        {header.selflearning.main_inputs.length === 0 ? (
          <p className="text-gray-500 text-center py-8">No self learning topics added yet.</p>
        ) : (
          <div className="space-y-4">
            {header.selflearning.main_inputs.map((mainInput, mainIndex) => (
              <div key={mainIndex} className="border border-gray-300 rounded-lg p-5 bg-white shadow-sm">
                {/* Main Input */}
                <div className="flex items-start space-x-4 mb-4">
                  <div className="flex-1">
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Main Topic {mainIndex + 1}
                    </label>
                    <input
                      type="text"
                      value={mainInput.main}
                      onChange={(e) => updateMainInput(mainIndex, e.target.value)}
                      className="input-custom"
                      placeholder="Enter main topic"
                    />
                  </div>
                  <button
                    type="button"
                    onClick={() => removeMainInput(mainIndex)}
                    className="text-red-600 hover:text-red-800 p-2 mt-6"
                    title="Remove main topic"
                  >
                    <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z" clipRule="evenodd" />
                    </svg>
                  </button>
                </div>

                {/* Internal Inputs */}
                <div className="ml-6 space-y-3">
                  <div className="flex justify-between items-center">
                    <h5 className="text-sm font-medium text-gray-600">Internal Resources (Text or Links)</h5>
                    <button
                      type="button"
                      onClick={() => addInternalInput(mainIndex)}
                      className="text-sm px-3 py-1.5 bg-sky-500 text-white rounded hover:bg-sky-600 transition-colors"
                    >
                      + Add Resource
                    </button>
                  </div>

                  {mainInput.internal.length === 0 ? (
                    <p className="text-gray-400 text-sm italic">No internal resources added yet.</p>
                  ) : (
                    <div className="space-y-2">
                      {mainInput.internal.map((internal, internalIndex) => (
                        <div key={internalIndex} className="flex items-center space-x-3 bg-gray-50 p-3 rounded border border-gray-200">
                          <input
                            type="text"
                            value={internal}
                            onChange={(e) => updateInternalInput(mainIndex, internalIndex, e.target.value)}
                            className="input-custom flex-1"
                            placeholder="Enter text or link"
                          />
                          <button
                            type="button"
                            onClick={() => removeInternalInput(mainIndex, internalIndex)}
                            className="text-red-600 hover:text-red-800 p-1"
                            title="Remove resource"
                          >
                            <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                              <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
                            </svg>
                          </button>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    )
  }

  // ============================================================================
  // MODEL OPERATIONS
  // ============================================================================

  const handleAddModel = async () => {
    // Auto-generate module/unit name based on current count and template
    const moduleNumber = models.length + 1
    const modelName = curriculumTemplate === '2022' ? `Unit ${moduleNumber}` : `Module ${moduleNumber}`

    try {
      const response = await fetch(`${API_BASE_URL}/course/${courseId}/syllabus/model`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          model_name: modelName,
          position: models.length
        })
      })

      if (!response.ok) throw new Error('Failed to create model')

      setSuccess(`${curriculumTemplate === '2022' ? 'Unit' : 'Module'} added successfully!`)
      setTimeout(() => setSuccess(''), 3000)
      await fetchSyllabus()
    } catch (err) {
      console.error('Error adding model:', err)
      setError(`Failed to add ${curriculumTemplate === '2022' ? 'unit' : 'module'}`)
    }
  }

  const handleUpdateModel = async (modelId, currentName, currentPosition) => {
    const newName = prompt('Enter new model name:', currentName)
    if (!newName || newName === currentName) return

    try {
      const response = await fetch(`${API_BASE_URL}/syllabus/model/${modelId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          model_name: newName,
          position: typeof currentPosition === 'number' ? currentPosition : models.findIndex(m => m.id === modelId)
        })
      })

      if (!response.ok) throw new Error('Failed to update model')

      setSuccess('Model updated successfully!')
      setTimeout(() => setSuccess(''), 3000)
      await fetchSyllabus()
    } catch (err) {
      console.error('Error updating model:', err)
      setError('Failed to update model')
    }
  }

  const handleDeleteModel = async (modelId) => {
    if (!window.confirm('Delete this model? This will also delete all its titles and topics.')) return

    try {
      const response = await fetch(`${API_BASE_URL}/syllabus/model/${modelId}`, {
        method: 'DELETE'
      })

      if (!response.ok) throw new Error('Failed to delete model')

      setSuccess('Model deleted successfully!')
      setTimeout(() => setSuccess(''), 3000)
      await fetchSyllabus()
    } catch (err) {
      console.error('Error deleting model:', err)
      setError('Failed to delete model')
    }
  }

  const toggleModel = (modelId) => {
    setExpandedModels(prev => ({ ...prev, [modelId]: !prev[modelId] }))
  }

  // ============================================================================
  // TITLE OPERATIONS
  // ============================================================================

  const handleAddTitle = async (modelId) => {
    const titleName = prompt('Enter title name:')
    if (!titleName) return
    
    const hours = prompt('Enter hours:', '0')
    const hoursNum = parseInt(hours) || 0

    try {
      const model = models.find(m => m.id === modelId)
      const response = await fetch(`${API_BASE_URL}/syllabus/model/${modelId}/title`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          title_name: titleName,
          hours: hoursNum,
          position: model?.titles?.length || 0
        })
      })

      if (!response.ok) throw new Error('Failed to create title')

      setSuccess('Title added successfully!')
      setTimeout(() => setSuccess(''), 3000)
      await fetchSyllabus()
    } catch (err) {
      console.error('Error adding title:', err)
      setError('Failed to add title')
    }
  }

  const handleUpdateTitle = async (titleId, currentName, currentHours, currentPosition) => {
    const newName = prompt('Enter new title name:', currentName)
    if (!newName) return
    
    const hours = prompt('Enter hours:', currentHours.toString())
    const hoursNum = parseInt(hours) || 0

    try {
      const response = await fetch(`${API_BASE_URL}/syllabus/title/${titleId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          title_name: newName,
          hours: hoursNum,
          position: typeof currentPosition === 'number' ? currentPosition : 0
        })
      })

      if (!response.ok) throw new Error('Failed to update title')

      setSuccess('Title updated successfully!')
      setTimeout(() => setSuccess(''), 3000)
      await fetchSyllabus()
    } catch (err) {
      console.error('Error updating title:', err)
      setError('Failed to update title')
    }
  }

  const handleDeleteTitle = async (titleId) => {
    if (!window.confirm('Delete this title? This will also delete all its topics.')) return

    try {
      const response = await fetch(`${API_BASE_URL}/syllabus/title/${titleId}`, {
        method: 'DELETE'
      })

      if (!response.ok) throw new Error('Failed to delete title')

      setSuccess('Title deleted successfully!')
      setTimeout(() => setSuccess(''), 3000)
      await fetchSyllabus()
    } catch (err) {
      console.error('Error deleting title:', err)
      setError('Failed to delete title')
    }
  }

  const toggleTitle = (titleId) => {
    setExpandedTitles(prev => ({ ...prev, [titleId]: !prev[titleId] }))
  }

  // ============================================================================
  // TOPIC OPERATIONS
  // ============================================================================

  const handleAddTopic = async (titleId) => {
    const topic = prompt('Enter topic:')
    if (!topic) return

    try {
      const allTitles = models.flatMap(m => m.titles || [])
      const title = allTitles.find(t => t.id === titleId)
      
      const response = await fetch(`${API_BASE_URL}/syllabus/title/${titleId}/topic`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          topic: topic,
          position: title?.topics?.length || 0
        })
      })

      if (!response.ok) throw new Error('Failed to create topic')

      setSuccess('Topic added successfully!')
      setTimeout(() => setSuccess(''), 3000)
      await fetchSyllabus()
    } catch (err) {
      console.error('Error adding topic:', err)
      setError('Failed to add topic')
    }
  }

  const handleUpdateTopic = async (topicId, currentTopic, currentPosition) => {
    const newTopic = prompt('Enter new topic:', currentTopic)
    if (!newTopic || newTopic === currentTopic) return

    try {
      const response = await fetch(`${API_BASE_URL}/syllabus/topic/${topicId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          topic: newTopic,
          position: typeof currentPosition === 'number' ? currentPosition : 0
        })
      })

      if (!response.ok) throw new Error('Failed to update topic')

      setSuccess('Topic updated successfully!')
      setTimeout(() => setSuccess(''), 3000)
      await fetchSyllabus()
    } catch (err) {
      console.error('Error updating topic:', err)
      setError('Failed to update topic')
    }
  }

  const handleDeleteTopic = async (topicId) => {
    if (!window.confirm('Delete this topic?')) return

    try {
      const response = await fetch(`${API_BASE_URL}/syllabus/topic/${topicId}`, {
        method: 'DELETE'
      })

      if (!response.ok) throw new Error('Failed to delete topic')

      setSuccess('Topic deleted successfully!')
      setTimeout(() => setSuccess(''), 3000)
      await fetchSyllabus()
    } catch (err) {
      console.error('Error deleting topic:', err)
      setError('Failed to delete topic')
    }
  }

  // ============================================================================
  // EXPERIMENT OPERATIONS (2022 Template)
  // ============================================================================

  const handleAddExperiment = async () => {
    const expNumber = experiments.length + 1
    const expName = `Experiment ${expNumber}`
    const hours = prompt(`Enter hours for Experiment ${expNumber}:`)
    if (!hours || isNaN(hours)) return

    try {
      const response = await fetch(`${API_BASE_URL}/course/${courseId}/experiments`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          experiment_number: expNumber,
          experiment_name: expName,
          hours: parseInt(hours),
          topics: []
        })
      })

      if (!response.ok) throw new Error('Failed to create experiment')

      setSuccess('Experiment added successfully!')
      setTimeout(() => setSuccess(''), 3000)
      await fetchSyllabus()
    } catch (err) {
      console.error('Error adding experiment:', err)
      setError('Failed to add experiment')
    }
  }

  const handleUpdateExperimentHours = async (expId, newHours) => {
    const exp = experiments.find(e => e.id === expId)
    if (!exp) return

    try {
      const response = await fetch(`${API_BASE_URL}/experiments/${expId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          experiment_number: exp.experiment_number,
          experiment_name: exp.experiment_name,
          hours: newHours,
          topics: exp.topics || []
        })
      })

      if (!response.ok) throw new Error('Failed to update experiment')

      setSuccess('Experiment hours updated successfully!')
      setTimeout(() => setSuccess(''), 3000)
      await fetchSyllabus()
    } catch (err) {
      console.error('Error updating experiment:', err)
      setError('Failed to update experiment')
    }
  }

  const handleUpdateExperimentTopics = async (expId, newTopics) => {
    const exp = experiments.find(e => e.id === expId)
    if (!exp) return

    try {
      const response = await fetch(`${API_BASE_URL}/experiments/${expId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          experiment_number: exp.experiment_number,
          experiment_name: exp.experiment_name,
          hours: exp.hours || 0,
          topics: newTopics
        })
      })

      if (!response.ok) throw new Error('Failed to update experiment topics')

      setSuccess('Topics updated successfully!')
      setTimeout(() => setSuccess(''), 3000)
      await fetchSyllabus()
    } catch (err) {
      console.error('Error updating topics:', err)
      setError('Failed to update topics')
    }
  }

  const handleDeleteExperiment = async (expId) => {
    if (!window.confirm('Delete this experiment? This will also delete all its topics.')) return

    try {
      const response = await fetch(`${API_BASE_URL}/experiments/${expId}`, {
        method: 'DELETE'
      })

      if (!response.ok) throw new Error('Failed to delete experiment')

      setSuccess('Experiment deleted successfully!')
      setTimeout(() => setSuccess(''), 3000)
      await fetchSyllabus()
    } catch (err) {
      console.error('Error deleting experiment:', err)
      setError('Failed to delete experiment')
    }
  }

  const handleAddExperimentTopic = async (expId) => {
    const topic = prompt('Enter topic:')
    if (!topic) return

    const exp = experiments.find(e => e.id === expId)
    if (!exp) {
      setError('Experiment not found')
      return
    }

    const updatedTopics = [...(exp.topics || []), topic]

    try {
      const response = await fetch(`${API_BASE_URL}/experiments/${expId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          experiment_number: exp.experiment_number,
          experiment_name: exp.experiment_name,
          hours: exp.hours || 0,
          topics: updatedTopics
        })
      })

      if (!response.ok) throw new Error('Failed to add topic')

      setSuccess('Topic added successfully!')
      setTimeout(() => setSuccess(''), 3000)
      await fetchSyllabus()
    } catch (err) {
      console.error('Error adding topic:', err)
      setError('Failed to add topic')
    }
  }

  const handleDeleteExperimentTopic = async (expId, topicIndex) => {
    if (!window.confirm('Delete this topic?')) return

    const exp = experiments.find(e => e.id === expId)
    if (!exp) return

    const updatedTopics = exp.topics.filter((_, idx) => idx !== topicIndex)

    try {
      const response = await fetch(`${API_BASE_URL}/experiments/${expId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          experiment_number: exp.experiment_number,
          experiment_name: exp.experiment_name,
          hours: exp.hours || 0,
          topics: updatedTopics
        })
      })

      if (!response.ok) throw new Error('Failed to delete topic')

      setSuccess('Topic deleted successfully!')
      setTimeout(() => setSuccess(''), 3000)
      await fetchSyllabus()
    } catch (err) {
      console.error('Error deleting topic:', err)
      setError('Failed to delete topic')
    }
  }

  const toggleExperiment = (expId) => {
    setExpandedExperiments(prev => ({ ...prev, [expId]: !prev[expId] }))
  }

  // ============================================================================
  // RENDER FUNCTIONS
  // ============================================================================

  const renderHeaderField = (field, label) => (
    <div className="mb-6">
      <div className="flex justify-between items-center mb-2">
        <label className="block text-sm font-medium text-gray-700">{label}</label>
        <button
          type="button"
          onClick={() => addHeaderItem(field)}
          className="btn-primary-custom"
        >
          + Add {label}
        </button>
      </div>
      {header[field].map((item, index) => (
        <div key={index} className="flex gap-2 mb-2">
          <input
            type="text"
            value={item}
            onChange={(e) => updateHeaderItem(field, index, e.target.value)}
            className="input-custom flex-1"
            placeholder={`Enter ${label.toLowerCase()}`}
          />
          <button
            type="button"
            onClick={() => removeHeaderItem(field, index)}
            className="px-4 py-2 bg-red-500 text-white rounded-lg hover:bg-red-600 transition-all"
          >
            Remove
          </button>
        </div>
      ))}
    </div>
  )

  const renderExperiments = () => (
    <div className="space-y-4">
      <div className="flex justify-between items-center mb-4">
        <h3 className="text-lg font-semibold">Course Experiments</h3>
        <button
          onClick={handleAddExperiment}
          className="btn-primary-custom"
        >
          + Add Experiment
        </button>
      </div>

      {experiments.length === 0 ? (
        <p className="text-gray-500">
          No experiments yet. Click "Add Experiment" to get started.
        </p>
      ) : (
        experiments.map((experiment) => (
          <div key={experiment.id} className="border border-gray-200 rounded-lg overflow-hidden bg-white shadow-sm">
            {/* Experiment Header */}
            <div className="bg-gray-50 px-4 py-3 flex justify-between items-center border-b border-gray-200">
              <div className="flex items-center gap-3 flex-1">
                <span className="font-semibold text-gray-700">{experiment.experiment_name}</span>
                <div className="flex items-center gap-2">
                  <label className="text-sm text-gray-600">Hours:</label>
                  <input
                    key={`hours-${experiment.id}-${experiment.hours}`}
                    type="number"
                    defaultValue={experiment.hours || 0}
                    onBlur={(e) => {
                      const inputValue = e.target.value.trim()
                      const newHours = inputValue === '' ? 0 : parseInt(inputValue, 10)
                      const currentHours = experiment.hours || 0
                      if (!isNaN(newHours) && newHours !== currentHours) {
                        handleUpdateExperimentHours(experiment.id, newHours)
                      }
                    }}
                    className="input-custom w-20"
                    placeholder="0"
                    min="0"
                  />
                </div>
              </div>
              <div className="flex gap-2">
                <button
                  onClick={() => toggleExperiment(experiment.id)}
                  className="px-3 py-1 text-sm bg-sky-500 text-white rounded hover:bg-sky-600 transition-all"
                >
                  {expandedExperiments[experiment.id] ? 'Hide Topics' : 'Show Topics'}
                </button>
                <button
                  onClick={() => handleDeleteExperiment(experiment.id)}
                  className="px-3 py-1 text-sm bg-red-500 text-white rounded hover:bg-red-600 transition-all"
                >
                  Delete
                </button>
              </div>
            </div>

            {/* Experiment Topics */}
            {expandedExperiments[experiment.id] && (
              <div className="p-4 bg-white">
                <div className="flex justify-between items-center mb-3">
                  <h4 className="text-sm font-semibold text-gray-700">Topics</h4>
                  <button
                    onClick={() => handleAddExperimentTopic(experiment.id)}
                    className="px-3 py-1 text-sm bg-green-500 text-white rounded hover:bg-green-600 transition-all"
                  >
                    + Add Topic
                  </button>
                </div>
                
                {(!experiment.topics || experiment.topics.length === 0) ? (
                  <p className="text-sm text-gray-500">No topics yet.</p>
                ) : (
                  <div className="space-y-2">
                    {experiment.topics.map((topic, topicIndex) => (
                      <div key={topicIndex} className="flex gap-2 items-center">
                        <input
                          type="text"
                          key={`topic-${experiment.id}-${topicIndex}-${topic}`}
                          defaultValue={topic}
                          onBlur={(e) => {
                            const nextValue = e.target.value
                            const currentValue = topic

                            if ((nextValue ?? '').trim() === (currentValue ?? '').trim()) return

                            const newTopics = [...(experiment.topics || [])]
                            newTopics[topicIndex] = nextValue
                            handleUpdateExperimentTopics(experiment.id, newTopics)
                          }}
                          className="input-custom flex-1"
                          placeholder={`Topic ${topicIndex + 1}`}
                        />
                        <button
                          onClick={() => handleDeleteExperimentTopic(experiment.id, topicIndex)}
                          className="px-3 py-1 text-sm bg-red-500 text-white rounded hover:bg-red-600 transition-all"
                        >
                          Remove
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
          </div>
        ))
      )}
    </div>
  )

  const renderModels = () => (
    <div className="space-y-4">
      <div className="flex justify-between items-center mb-4">
        <h3 className="text-lg font-semibold">
          {curriculumTemplate === '2022' ? 'Syllabus Units' : 'Syllabus Modules'}
        </h3>
        <button
          onClick={handleAddModel}
          className="btn-primary-custom"
        >
          {curriculumTemplate === '2022' ? '+ Add Unit' : '+ Add Module'}
        </button>
      </div>

      {models.length === 0 ? (
        <p className="text-gray-500">
          No {curriculumTemplate === '2022' ? 'units' : 'models'} yet. Click "Add {curriculumTemplate === '2022' ? 'Unit' : 'Module'}" to get started.
        </p>
      ) : (
        models.map((model) => (
          <div key={model.id} className="border border-gray-200 rounded-lg overflow-hidden bg-white shadow-sm">
            {/* Model Header */}
            <div className="bg-gray-50 px-4 py-3 flex justify-between items-center border-b border-gray-200">
              <div className="flex items-center gap-3">
                <button
                  onClick={() => toggleModel(model.id)}
                  className="text-gray-600 hover:text-gray-800"
                >
                  {expandedModels[model.id] ? '▼' : '▶'}
                </button>
                <span className="font-semibold text-lg">{model.model_name}</span>
              </div>
              <div className="flex gap-2">
                <button
                  onClick={() => handleUpdateModel(model.id, model.model_name, model.position)}
                  className="px-3 py-1 bg-sky-500 text-white text-sm rounded hover:bg-sky-600 transition-colors"
                >
                  Edit
                </button>
                <button
                  onClick={() => handleDeleteModel(model.id)}
                  className="px-3 py-1 bg-red-500 text-white text-sm rounded hover:bg-red-600 transition-colors"
                >
                  Delete
                </button>
              </div>
            </div>

            {/* Expanded: Show Titles */}
            {expandedModels[model.id] && (
              <div className="p-4 bg-white">
                <div className="flex justify-between items-center mb-3">
                  <h4 className="font-medium text-gray-700">Titles</h4>
                  <button
                    onClick={() => handleAddTitle(model.id)}
                    className="px-3 py-1.5 bg-green-500 text-white text-sm rounded hover:bg-green-600 transition-colors"
                  >
                    + Add Title
                  </button>
                </div>

                {(!model.titles || model.titles.length === 0) ? (
                  <p className="text-gray-500 text-sm">No titles yet.</p>
                ) : (
                  <div className="space-y-2">
                    {model.titles.map((title) => (
                      <div key={title.id} className="border border-gray-200 rounded-lg overflow-hidden ml-4 bg-white">
                        {/* Title Header */}
                        <div className="bg-gray-50 px-3 py-2 flex justify-between items-center">
                          <div className="flex items-center gap-2">
                            <button
                              onClick={() => toggleTitle(title.id)}
                              className="text-gray-600 hover:text-gray-800 text-sm"
                            >
                              {expandedTitles[title.id] ? '▼' : '▶'}
                            </button>
                            <span className="font-medium">{title.title_name}</span>
                            <span className="text-sm text-gray-600">({title.hours}h)</span>
                          </div>
                          <div className="flex gap-2">
                            <button
                              onClick={() => handleUpdateTitle(title.id, title.title_name, title.hours, title.position)}
                              className="px-2 py-1 bg-sky-500 text-white text-xs rounded hover:bg-sky-600 transition-colors"
                            >
                              Edit
                            </button>
                            <button
                              onClick={() => handleDeleteTitle(title.id)}
                              className="px-2 py-1 bg-red-500 text-white text-xs rounded hover:bg-red-600 transition-colors"
                            >
                              Delete
                            </button>
                          </div>
                        </div>

                        {/* Expanded: Show Topics */}
                        {expandedTitles[title.id] && (
                          <div className="p-3 bg-white border-t border-gray-100">
                            <div className="flex justify-between items-center mb-2">
                              <h5 className="text-sm font-medium text-gray-700">Topics</h5>
                              <button
                                onClick={() => handleAddTopic(title.id)}
                                className="px-2 py-1 bg-green-500 text-white text-xs rounded hover:bg-green-600 transition-colors"
                              >
                                + Add Topic
                              </button>
                            </div>

                            {(!title.topics || title.topics.length === 0) ? (
                              <p className="text-gray-500 text-xs">No topics yet.</p>
                            ) : (
                              <ul className="space-y-1.5">
                                {title.topics.map((topic) => (
                                  <li key={topic.id} className="flex justify-between items-center bg-gray-50 px-3 py-2 rounded text-sm border border-gray-100">
                                    <span>{topic.topic}</span>
                                    <div className="flex gap-2">
                                      <button
                                        onClick={() => handleUpdateTopic(topic.id, topic.topic, topic.position)}
                                        className="px-2 py-0.5 bg-sky-500 text-white text-xs rounded hover:bg-sky-600 transition-colors"
                                      >
                                        Edit
                                      </button>
                                      <button
                                        onClick={() => handleDeleteTopic(topic.id)}
                                        className="px-2 py-0.5 bg-red-500 text-white text-xs rounded hover:bg-red-600 transition-colors"
                                      >
                                        Delete
                                      </button>
                                    </div>
                                  </li>
                                ))}
                              </ul>
                            )}
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
          </div>
        ))
      )}
    </div>
  )

  if (loading) {
    return (
      <MainLayout title="Course Syllabus" subtitle="Loading...">
        <div className="flex justify-center items-center py-20">
          <div className="text-center">
            <svg className="animate-spin h-12 w-12 text-primary mx-auto mb-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            <p className="text-gray-600">Loading syllabus...</p>
          </div>
        </div>
      </MainLayout>
    )
  }

  return (
    <MainLayout 
      title={courseInfo ? `Course Syllabus – ${courseInfo.course_code} ${courseInfo.course_name}` : 'Course Syllabus'}
      subtitle="Manage course outcomes, modules, and resources"
      actions={
        <button
          onClick={() => navigate(-1)}
          className="btn-secondary-custom flex items-center space-x-2"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
          </svg>
          <span>Back</span>
        </button>
      }
    >
      <div className="max-w-7xl mx-auto space-y-6">

        {/* Messages */}
        {error && (
          <div className="flex items-start space-x-3 p-4 bg-red-50 border border-red-200 rounded-lg">
            <svg className="w-5 h-5 text-red-600 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
            </svg>
            <p className="text-sm font-medium text-red-600">{error}</p>
          </div>
        )}

        {success && (
          <div className="flex items-start space-x-3 p-4 bg-green-50 border border-green-200 rounded-lg">
            <svg className="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
            </svg>
            <p className="text-sm font-medium text-green-600">{success}</p>
          </div>
        )}

        {/* Tabs */}
        <div className="card-custom overflow-hidden">
          <nav className="flex justify-center overflow-x-auto border-b border-gray-200">
            {[...(curriculumTemplate === '2022' ? ['objectives'] : []),
              'outcomes', curriculumTemplate === '2022' ? 'units' : 'modules',
              ...(curriculumTemplate === '2022' ? ['experiments'] : ['teamwork', 'selflearning', 'prerequisites']),
              'references', 'textbook_references'].map((tab) => (
              <button
                key={tab}
                onClick={() => setActiveTab(tab)}
                className={`px-8 py-4 font-medium text-sm whitespace-nowrap border-b-2 transition-colors ${
                  activeTab === tab
                    ? 'border-primary text-primary'
                    : 'border-transparent text-gray-500 hover:text-gray-700'
                }`}
              >
                {tab === 'objectives' ? 'Objectives' :
                 tab === 'teamwork' ? 'Team Work' : 
                 tab === 'selflearning' ? 'Self Learning' : 
                 tab === 'textbook_references' ? 'Textbook References' :
                 tab === 'units' ? 'Units' :
                 tab === 'modules' ? 'Modules' :
                 tab.charAt(0).toUpperCase() + tab.slice(1)}
              </button>
            ))}
          </nav>
        </div>

        {/* Tab Content */}
        <div className="card-custom p-6">
          {activeTab === 'objectives' && curriculumTemplate === '2022' && (
            <form onSubmit={handleSaveHeader}>
              {renderHeaderField('objectives', 'Objectives')}
              <button
                type="submit"
                className="btn-primary-custom"
              >
                Save Objectives
              </button>
            </form>
          )}

          {activeTab === 'outcomes' && (
            <form onSubmit={handleSaveHeader}>
              {renderHeaderField('outcomes', 'Outcomes')}
              <button
                type="submit"
                className="btn-primary-custom"
              >
                Save Outcomes
              </button>
            </form>
          )}

          {(activeTab === 'modules' || activeTab === 'units') && renderModels()}

          {activeTab === 'experiments' && curriculumTemplate === '2022' && renderExperiments()}

          {activeTab === 'teamwork' && curriculumTemplate !== '2022' && (
            <form onSubmit={handleSaveHeader}>
              {renderTeamwork()}
              <button
                type="submit"
                className="btn-primary-custom mt-6"
              >
                Save Team Work
              </button>
            </form>
          )}

          {activeTab === 'selflearning' && curriculumTemplate !== '2022' && (
            <form onSubmit={handleSaveHeader}>
              {renderSelfLearning()}
              <button
                type="submit"
                className="btn-primary-custom mt-6"
              >
                Save Self Learning
              </button>
            </form>
          )}

          {activeTab === 'references' && (
            <form onSubmit={handleSaveHeader}>
              {renderHeaderField('reference_list', 'References')}
              <button
                type="submit"
                className="btn-primary-custom"
              >
                Save References
              </button>
            </form>
          )}

          {activeTab === 'textbook_references' && (
            <form onSubmit={handleSaveHeader}>
              {renderHeaderField('textbook_reference_list', 'Textbook References')}
              <button
                type="submit"
                className="btn-primary-custom"
              >
                Save Textbook References
              </button>
            </form>
          )}

          {activeTab === 'prerequisites' && curriculumTemplate !== '2022' && (
            <form onSubmit={handleSaveHeader}>
              {renderHeaderField('prerequisites', 'Prerequisites')}
              <button
                type="submit"
                className="btn-primary-custom"
              >
                Save Prerequisites
              </button>
            </form>
          )}
        </div>
      </div>
    </MainLayout>
  )
}

export default SyllabusPage
