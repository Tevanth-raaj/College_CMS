import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import MainLayout from '../../components/MainLayout'
import { API_BASE_URL } from '../../config'

function ClusterManagementPage() {
  const navigate = useNavigate()
  const [clusters, setClusters] = useState([])
  const [curriculum, setCurriculum] = useState([])
  const [availableCurriculum, setAvailableCurriculum] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [showCreateForm, setShowCreateForm] = useState(false)
  const [newCluster, setNewCluster] = useState({ name: '', description: '' })
  const [selectedCluster, setSelectedCluster] = useState(null)
  const [clusterDepartments, setClusterDepartments] = useState([])
  const [showAddDepartmentModal, setShowAddDepartmentModal] = useState(false)
  const [selectedDepartmentId, setSelectedDepartmentId] = useState('')

  useEffect(() => {
    fetchClusters()
    fetchCurriculum()
  }, [])

  const fetchClusters = async () => {
    try {
      setLoading(true)
      const response = await fetch(`${API_BASE_URL}/clusters`)
      if (!response.ok) {
        throw new Error('Failed to fetch clusters')
      }
      const data = await response.json()
      setClusters(data || [])
      setError('')
    } catch (err) {
      console.error('Error fetching clusters:', err)
      setError('Failed to load clusters')
    } finally {
      setLoading(false)
    }
  }

  const fetchCurriculum = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/curriculum`)
      if (!response.ok) {
        throw new Error('Failed to fetch curriculum')
      }
      const data = await response.json()
      setCurriculum(data || [])
    } catch (err) {
      console.error('Error fetching curriculum:', err)
    }
  }

  const fetchAvailableDepartments = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/clusters/available-departments`)
      if (!response.ok) {
        throw new Error('Failed to fetch available departments')
      }
      const data = await response.json()
      setAvailableCurriculum(data || [])
    } catch (err) {
      console.error('Error fetching available departments:', err)
      setAvailableCurriculum([])
    }
  }

  const fetchClusterDepartments = async (clusterId) => {
    try {
      const response = await fetch(`${API_BASE_URL}/cluster/${clusterId}/departments`)
      if (!response.ok) {
        throw new Error('Failed to fetch cluster departments')
      }
      const data = await response.json()
      setClusterDepartments(data || [])
    } catch (err) {
      console.error('Error fetching cluster departments:', err)
      setClusterDepartments([])
    }
  }

  const handleCreateCluster = async (e) => {
    e.preventDefault()
    
    if (!newCluster.name.trim()) {
      setError('Cluster name is required')
      return
    }

    try {
      const response = await fetch(`${API_BASE_URL}/clusters`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(newCluster),
      })

      if (!response.ok) {
        throw new Error('Failed to create cluster')
      }

      setNewCluster({ name: '', description: '' })
      setShowCreateForm(false)
      setSuccess('Cluster created successfully!')
      setTimeout(() => setSuccess(''), 3000)
      fetchClusters()
    } catch (err) {
      console.error('Error creating cluster:', err)
      setError('Failed to create cluster')
    }
  }

  const handleDeleteCluster = async (clusterId) => {
    if (!window.confirm('Are you sure you want to delete this cluster? All department associations will be removed.')) {
      return
    }

    try {
      const response = await fetch(`${API_BASE_URL}/cluster/${clusterId}`, {
        method: 'DELETE',
      })

      if (!response.ok) {
        throw new Error('Failed to delete cluster')
      }

      setSuccess('Cluster deleted successfully!')
      setTimeout(() => setSuccess(''), 3000)
      fetchClusters()
      if (selectedCluster?.id === clusterId) {
        setSelectedCluster(null)
        setClusterDepartments([])
      }
    } catch (err) {
      console.error('Error deleting cluster:', err)
      setError('Failed to delete cluster')
    }
  }

  const handleViewCluster = (cluster) => {
    setSelectedCluster(cluster)
    fetchClusterDepartments(cluster.id)
  }

  const handleOpenAddDepartmentModal = () => {
    fetchAvailableDepartments()
    setShowAddDepartmentModal(true)
  }

  const handleAddDepartment = async () => {
    if (!selectedDepartmentId) {
      setError('Please select a department')
      return
    }

    try {
      const response = await fetch(`${API_BASE_URL}/cluster/${selectedCluster.id}/department`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ department_id: parseInt(selectedDepartmentId) }),
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to add department')
      }

      setShowAddDepartmentModal(false)
      setSelectedDepartmentId('')
      setSuccess('Department added to cluster successfully!')
      setTimeout(() => setSuccess(''), 3000)
      fetchClusterDepartments(selectedCluster.id)
      fetchAvailableDepartments()
    } catch (err) {
      console.error('Error adding department:', err)
      setError(err.message || 'Failed to add department to cluster')
    }
  }

  const handleRemoveDepartment = async (deptId) => {
    if (!window.confirm('Are you sure you want to remove this department from the cluster?')) {
      return
    }

    try {
      const response = await fetch(`${API_BASE_URL}/cluster/${selectedCluster.id}/department/${deptId}`, {
        method: 'DELETE',
      })

      if (!response.ok) {
        throw new Error('Failed to remove department')
      }

      setSuccess('Department removed from cluster successfully!')
      setTimeout(() => setSuccess(''), 3000)
      fetchClusterDepartments(selectedCluster.id)
      fetchAvailableDepartments()
    } catch (err) {
      console.error('Error removing department:', err)
      setError('Failed to remove department from cluster')
    }
  }

  const getDepartmentName = (dept) => {
    // If dept has a name property from backend, use it
    if (dept && dept.name) {
      return dept.name
    }
    // Otherwise look up by curriculum_id or id
    const regId = dept.curriculum_id || dept.id
    const reg = curriculum.find(r => r.id === regId)
    return reg ? reg.name : `Department ${regId}`
  }

  if (loading) {
    return (
      <MainLayout title="Cluster Management" subtitle="Loading...">
        <div className="flex justify-center items-center py-20">
          <div className="text-center">
            <svg className="animate-spin h-12 w-12 text-blue-600 mx-auto mb-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            <p className="text-gray-600">Loading clusters...</p>
          </div>
        </div>
      </MainLayout>
    )
  }

  return (
    <MainLayout 
      title="Cluster Management" 
      subtitle="Manage department clusters for shared content"
      actions={
        <div className="flex items-center space-x-3">
          <button
            onClick={() => navigate('/dashboard')}
            className="btn-secondary-custom flex items-center space-x-2"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
            </svg>
            <span>Back to Dashboard</span>
          </button>
          <button
            onClick={() => setShowCreateForm(!showCreateForm)}
            className="btn-primary-custom flex items-center space-x-2"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            <span>{showCreateForm ? 'Cancel' : 'Create Cluster'}</span>
          </button>
        </div>
      }
    >
      <div className="space-y-6">
        {/* Messages */}
        {error && (
          <div className="flex items-center justify-between p-4 bg-red-50 border border-red-200 rounded-lg space-x-3">
            <div className="flex items-center space-x-2">
              <svg className="w-5 h-5 text-red-600 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
              </svg>
              <p className="text-sm font-medium text-red-600">{error}</p>
            </div>
            {/* Retry button */}
            <button
              onClick={() => window.location.reload()}
              className="px-3 py-1.5 text-sm font-medium text-red-700 bg-red-50 border border-red-200 rounded-md hover:bg-red-100 hover:text-red-800 focus:outline-none focus:ring-2 focus:ring-red-300 transition"
            >
              Retry
            </button>
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

        {/* Create Form */}
        {showCreateForm && (
          <div className="card-custom p-6">
            <h2 className="text-lg font-bold text-gray-900 mb-6">Create New Cluster</h2>
            <form onSubmit={handleCreateCluster} className="space-y-5">
              <div>
                <label className="block text-sm font-semibold text-gray-700 mb-2">Cluster Name</label>
                <input
                  type="text"
                  value={newCluster.name}
                  onChange={(e) => setNewCluster({ ...newCluster, name: e.target.value })}
                  placeholder="e.g., Engineering Cluster"
                  required
                  className="input-custom"
                />
              </div>

              <div>
                <label className="block text-sm font-semibold text-gray-700 mb-2">Description (Optional)</label>
                <textarea
                  value={newCluster.description}
                  onChange={(e) => setNewCluster({ ...newCluster, description: e.target.value })}
                  placeholder="Brief description of this cluster"
                  rows="3"
                  className="input-custom resize-none"
                />
              </div>

              <div className="flex gap-3 justify-end">
                <button
                  type="button"
                  onClick={() => {
                    setShowCreateForm(false)
                    setNewCluster({ name: '', description: '' })
                  }}
                  className="btn-secondary-custom"
                >
                  Cancel
                </button>
                <button type="submit" className="btn-primary-custom">
                  Create Cluster
                </button>
              </div>
            </form>
          </div>
        )}

        {/* Two Column Layout */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Clusters List */}
          <div className="card-custom p-6">
            <h2 className="text-lg font-bold text-gray-900 mb-5">All Clusters</h2>
            
            {clusters.length === 0 ? (
              <div className="text-center py-12 text-gray-500">
                <svg className="w-16 h-16 text-gray-300 mx-auto mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
                </svg>
                <p className="text-sm">No clusters yet. Click <button type="button" onClick={() => setShowCreateForm(true)} className="text-primary hover:underline">Create Cluster</button> to start.</p>
              </div>
            ) : (
              <div className="space-y-3">
                {clusters.map(cluster => (
                  <div
                    key={cluster.id}
                    className={`p-4 rounded-lg border-2 cursor-pointer transition-all ${
                      selectedCluster?.id === cluster.id
                        ? 'border-blue-500 bg-blue-50'
                        : 'border-gray-200 hover:border-blue-300 hover:bg-gray-50'
                    }`}
                    onClick={() => handleViewCluster(cluster)}
                  >
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <h3 className="font-bold text-gray-900">{cluster.name}</h3>
                        {cluster.description && (
                          <p className="text-sm text-gray-600 mt-1">{cluster.description}</p>
                        )}
                      </div>
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          handleDeleteCluster(cluster.id)
                        }}
                        className="ml-3 p-2 text-red-600 hover:bg-red-50 rounded-lg transition-colors"
                        title="Delete Cluster"
                      >
                        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                        </svg>
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Cluster Details */}
          <div className="card-custom p-6">
            <div className="flex items-center justify-between mb-5">
              <h2 className="text-lg font-bold text-gray-900">
                {selectedCluster ? `Departments in ${selectedCluster.name}` : 'Select a Cluster'}
              </h2>
              {selectedCluster && (
                <button
                  onClick={handleOpenAddDepartmentModal}
                  className="btn-primary-custom text-sm flex items-center space-x-2"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                  </svg>
                  <span>Add Department</span>
                </button>
              )}
            </div>

            {!selectedCluster ? (
              <div className="text-center py-12 text-gray-500">
                <svg className="w-16 h-16 text-gray-300 mx-auto mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zm0 0h12a2 2 0 002-2v-4a2 2 0 00-2-2h-2.343M11 7.343l1.657-1.657a2 2 0 012.828 0l2.829 2.829a2 2 0 010 2.828l-8.486 8.485M7 17h.01" />
                </svg>
                <p className="text-sm">Select a cluster to view its departments</p>
              </div>
            ) : clusterDepartments.length === 0 ? (
              <div className="text-center py-12 text-gray-500">
                <svg className="w-16 h-16 text-gray-300 mx-auto mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
                </svg>
                <p className="text-sm">No departments in this cluster yet</p>
              </div>
            ) : (
              <div className="space-y-4">
                {/* Group by curriculum template */}
                {Object.entries(
                  clusterDepartments.reduce((groups, dept) => {
                    const template = dept.curriculum_template || '2026'
                    if (!groups[template]) groups[template] = []
                    groups[template].push(dept)
                    return groups
                  }, {})
                ).map(([template, depts]) => (
                  <div key={template} className="space-y-2">
                    <div className="flex items-center space-x-2 px-2">
                      <div className={`px-3 py-1 rounded-full text-xs font-semibold ${
                        template === '2022' 
                          ? 'bg-purple-100 text-purple-700' 
                          : 'bg-green-100 text-green-700'
                      }`}>
                        {template} Template
                      </div>
                      <div className="flex-1 h-px bg-gray-200"></div>
                    </div>
                    {depts.map(dept => (
                      <div
                        key={dept.id}
                        className="flex items-center justify-between p-3 bg-gray-50 rounded-lg border border-gray-200"
                      >
                        <div className="flex items-center space-x-3">
                          <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${
                            template === '2022' 
                              ? 'bg-purple-100' 
                              : 'bg-blue-100'
                          }`}>
                            <svg className={`w-5 h-5 ${
                              template === '2022' 
                                ? 'text-purple-600' 
                                : 'text-blue-600'
                            }`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
                            </svg>
                          </div>
                          <div>
                            <p className="font-semibold text-gray-900">{getDepartmentName(dept)}</p>
                            <p className="text-xs text-gray-500">Curriculum ID: {dept.curriculum_id || dept.id}</p>
                          </div>
                        </div>
                        <button
                          onClick={() => handleRemoveDepartment(dept.curriculum_id || dept.id)}
                          className="p-2 text-red-600 hover:bg-red-50 rounded-lg transition-colors"
                          title="Remove from cluster"
                        >
                          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                          </svg>
                        </button>
                      </div>
                    ))}
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Add Department Modal */}
        {showAddDepartmentModal && (
          <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 p-4" onClick={() => setShowAddDepartmentModal(false)}>
            <div className="bg-background rounded-2xl shadow-2xl max-w-md w-full" onClick={(e) => e.stopPropagation()}>
              <div className="bg-gradient-to-r from-blue-600 to-blue-700 text-white px-8 py-5 flex items-center justify-between rounded-t-2xl">
                <div>
                  <h3 className="text-2xl font-bold mb-1">Add Department</h3>
                  <p className="text-sm text-blue-100">Add a department to {selectedCluster?.name}</p>
                </div>
                <button 
                  onClick={() => setShowAddDepartmentModal(false)}
                  className="text-white hover:bg-white/20 rounded-xl p-2.5 transition-all"
                >
                  <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
              
              <div className="p-8 space-y-5">
                <div>
                  <label className="block text-sm font-semibold text-gray-700 mb-2">Select Department</label>
                  <select
                    value={selectedDepartmentId}
                    onChange={(e) => setSelectedDepartmentId(e.target.value)}
                    className="input-custom"
                  >
                    <option value="">Choose a department...</option>
                    {availableCurriculum.length === 0 ? (
                      <option value="" disabled>No available departments</option>
                    ) : (
                      availableCurriculum.map(reg => (
                        <option key={reg.id} value={reg.id}>
                          {reg.name}
                        </option>
                      ))
                    )}
                  </select>
                </div>

                <div className="flex gap-3 justify-end pt-2">
                  <button
                    type="button"
                    onClick={() => {
                      setShowAddDepartmentModal(false)
                      setSelectedDepartmentId('')
                    }}
                    className="btn-secondary-custom"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={handleAddDepartment}
                    className="btn-primary-custom"
                  >
                    Add Department
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </MainLayout>
  )
}

export default ClusterManagementPage
