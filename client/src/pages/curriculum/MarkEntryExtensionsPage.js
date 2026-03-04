import React, { useEffect, useState } from 'react'
import MainLayout from '../../components/MainLayout'
import { API_BASE_URL } from '../../config'

function MarkEntryExtensionsPage() {
  const userRole = (localStorage.getItem('userRole') || '').toLowerCase()
  const username = localStorage.getItem('username')

  const [requests, setRequests] = useState([])
  const [loading, setLoading] = useState(false)
  const [message, setMessage] = useState({ type: '', text: '' })

  const [form, setForm] = useState({
    window_id: '',
    course_id: '',
    teacher_id: localStorage.getItem('teacherId') || '',
    reason: '',
    requested_end_at: '',
    semester: '',
    exam_type: 'PT1',
  })

  const canRequest = userRole === 'teacher' || userRole === 'hod'
  const canApprove = userRole === 'admin' || userRole === 'coe'

  const loadRequests = async () => {
    setLoading(true)
    setMessage({ type: '', text: '' })
    try {
      const params = new URLSearchParams()
      if (!canApprove) {
        params.set('requester_username', username || '')
        params.set('role', userRole)
      }
      const response = await fetch(`${API_BASE_URL}/mark-entry/extensions?${params.toString()}`)
      const data = await response.json()
      if (!response.ok) {
        throw new Error(data.error || data.message || 'Failed to fetch requests')
      }
      setRequests(Array.isArray(data) ? data : [])
    } catch (err) {
      setRequests([])
      setMessage({ type: 'error', text: err.message })
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadRequests()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const submitRequest = async (event) => {
    event.preventDefault()
    setMessage({ type: '', text: '' })
    try {
      const payload = {
        ...form,
        window_id: Number(form.window_id),
        course_id: Number(form.course_id),
        requester_role: userRole,
        requester_username: username,
        semester: form.semester ? Number(form.semester) : null,
      }
      const response = await fetch(`${API_BASE_URL}/mark-entry/extensions/request`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
      const data = await response.json()
      if (!response.ok) {
        throw new Error(data.error || data.message || 'Failed to submit request')
      }
      setMessage({ type: 'success', text: 'Extension request submitted.' })
      setForm((previous) => ({ ...previous, reason: '', requested_end_at: '' }))
      loadRequests()
    } catch (err) {
      setMessage({ type: 'error', text: err.message })
    }
  }

  const handleDecision = async (requestId, action) => {
    const approved_end_at = action === 'approve' ? window.prompt('Enter approved end datetime (YYYY-MM-DDTHH:mm):') : ''
    const rejection_reason = action === 'reject' ? window.prompt('Enter rejection reason:') : ''

    try {
      const response = await fetch(`${API_BASE_URL}/mark-entry/extensions/${requestId}/${action}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          approver_username: username,
          approver_role: userRole,
          approved_end_at,
          rejection_reason,
        }),
      })
      const data = await response.json()
      if (!response.ok) {
        throw new Error(data.error || data.message || `Failed to ${action} request`)
      }
      setMessage({ type: 'success', text: `Request ${action}d successfully.` })
      loadRequests()
    } catch (err) {
      setMessage({ type: 'error', text: err.message })
    }
  }

  return (
    <MainLayout
      title="Mark Entry Extensions"
      subtitle="Request window extension and manage approvals"
      actions={<button className="px-3 py-2 text-sm bg-primary text-white rounded-lg" onClick={loadRequests}>Refresh</button>}
    >
      <div className="space-y-4">
        {message.text && (
          <div className={`border rounded-lg p-3 text-sm ${message.type === 'error' ? 'bg-red-50 border-red-200 text-red-700' : 'bg-green-50 border-green-200 text-green-700'}`}>
            {message.text}
          </div>
        )}

        {canRequest && (
          <form onSubmit={submitRequest} className="bg-white border rounded-lg p-4 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3">
            <Input label="Window ID" value={form.window_id} onChange={(value) => setForm((previous) => ({ ...previous, window_id: value }))} required />
            <Input label="Course ID" value={form.course_id} onChange={(value) => setForm((previous) => ({ ...previous, course_id: value }))} required />
            <Input label="Teacher ID" value={form.teacher_id} onChange={(value) => setForm((previous) => ({ ...previous, teacher_id: value }))} required />
            <Input label="Requested End" type="datetime-local" value={form.requested_end_at} onChange={(value) => setForm((previous) => ({ ...previous, requested_end_at: value }))} required />
            <Input label="Semester" value={form.semester} onChange={(value) => setForm((previous) => ({ ...previous, semester: value }))} />
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">Exam Type</label>
              <select value={form.exam_type} onChange={(event) => setForm((previous) => ({ ...previous, exam_type: event.target.value }))} className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm">
                <option value="PT1">PT1</option>
                <option value="PT2">PT2</option>
                <option value="Model">Model</option>
                <option value="EndSem">EndSem</option>
              </select>
            </div>
            <div className="md:col-span-2">
              <label className="block text-xs font-medium text-gray-600 mb-1">Reason</label>
              <input
                type="text"
                value={form.reason}
                onChange={(event) => setForm((previous) => ({ ...previous, reason: event.target.value }))}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm"
                required
              />
            </div>
            <div className="flex items-end">
              <button type="submit" className="w-full bg-primary text-white px-4 py-2 rounded-lg text-sm">Submit Request</button>
            </div>
          </form>
        )}

        <div className="bg-white border rounded-lg overflow-hidden">
          <div className="overflow-x-auto">
            <table className="min-w-full text-sm">
              <thead className="bg-gray-50">
                <tr>
                  <th className="text-left px-3 py-2">Course</th>
                  <th className="text-left px-3 py-2">Teacher</th>
                  <th className="text-left px-3 py-2">Requester</th>
                  <th className="text-left px-3 py-2">Requested End</th>
                  <th className="text-left px-3 py-2">Status</th>
                  {canApprove && <th className="text-left px-3 py-2">Action</th>}
                </tr>
              </thead>
              <tbody>
                {loading ? (
                  <tr><td colSpan={canApprove ? 6 : 5} className="px-3 py-6 text-center text-gray-500">Loading...</td></tr>
                ) : requests.length === 0 ? (
                  <tr><td colSpan={canApprove ? 6 : 5} className="px-3 py-6 text-center text-gray-500">No requests available.</td></tr>
                ) : requests.map((request) => (
                  <tr key={request.id} className="border-t">
                    <td className="px-3 py-2">
                      <div className="font-medium text-gray-900">{request.course_code}</div>
                      <div className="text-gray-500">{request.course_name}</div>
                    </td>
                    <td className="px-3 py-2">
                      <div className="font-medium text-gray-900">{request.teacher_name}</div>
                      <div className="text-gray-500">{request.teacher_id}</div>
                    </td>
                    <td className="px-3 py-2">
                      <div className="font-medium text-gray-900">{request.requester_username}</div>
                      <div className="text-gray-500">{request.requester_role}</div>
                    </td>
                    <td className="px-3 py-2">{request.requested_end_at ? new Date(request.requested_end_at).toLocaleString() : '-'}</td>
                    <td className="px-3 py-2">
                      <span className={`px-2 py-1 rounded text-xs ${request.status === 'approved' ? 'bg-green-100 text-green-700' : request.status === 'rejected' ? 'bg-red-100 text-red-700' : 'bg-yellow-100 text-yellow-700'}`}>
                        {request.status}
                      </span>
                    </td>
                    {canApprove && (
                      <td className="px-3 py-2">
                        {request.status === 'pending' ? (
                          <div className="flex gap-2">
                            <button onClick={() => handleDecision(request.id, 'approve')} className="px-2 py-1 rounded bg-green-600 text-white text-xs">Approve</button>
                            <button onClick={() => handleDecision(request.id, 'reject')} className="px-2 py-1 rounded bg-red-600 text-white text-xs">Reject</button>
                          </div>
                        ) : (
                          <span className="text-gray-500 text-xs">Processed</span>
                        )}
                      </td>
                    )}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </MainLayout>
  )
}

function Input({ label, value, onChange, type = 'text', required = false }) {
  return (
    <div>
      <label className="block text-xs font-medium text-gray-600 mb-1">{label}</label>
      <input
        type={type}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm"
        required={required}
      />
    </div>
  )
}

export default MarkEntryExtensionsPage