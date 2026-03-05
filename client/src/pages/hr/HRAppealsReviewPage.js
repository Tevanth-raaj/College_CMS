import React, { useState, useEffect } from 'react';
import MainLayout from '../../components/MainLayout';
import { API_BASE_URL } from '../../config';

const HRAppealsReviewPage = () => {
  const [appeals, setAppeals] = useState([]);
  const [loading, setLoading] = useState(true);
  const [message, setMessage] = useState({ type: '', text: '' });
  const [statusFilter, setStatusFilter] = useState('pending');
  
  // Accept modal state
  const [showAcceptModal, setShowAcceptModal] = useState(false);
  const [selectedAppeal, setSelectedAppeal] = useState(null);
  const [currentAllocations, setCurrentAllocations] = useState([]);
  const [newAllocations, setNewAllocations] = useState([]);
  const [hrMessage, setHrMessage] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  
  // Window period state
  const [windowStatus, setWindowStatus] = useState({
    isOpen: false,
    startDate: null,
    endDate: null,
    academicYear: ''
  });

  useEffect(() => {
    fetchWindowPeriod();
    fetchAppeals();
  }, [statusFilter]);

  const fetchWindowPeriod = async () => {
    try {
      const calendarResponse = await fetch(`${API_BASE_URL}/academic-calendar/current`);
      if (calendarResponse.ok) {
        const calendarData = await calendarResponse.json();
        const windowResponse = await fetch(`${API_BASE_URL}/teachers/course-window/${calendarData.academic_year}`);
        if (windowResponse.ok) {
          const windowData = await windowResponse.json();
          const now = new Date();
          const start = new Date(windowData.preference_start_date);
          const end = new Date(windowData.preference_end_date);
          
          setWindowStatus({
            isOpen: now >= start && now <= end,
            startDate: start,
            endDate: end,
            academicYear: calendarData.academic_year
          });
        }
      }
    } catch (error) {
      console.error('Error fetching window period:', error);
    }
  };

  const fetchAppeals = async () => {
    try {
      setLoading(true);
      const url = `${API_BASE_URL}/hr/appeals?status=${statusFilter}`;
      const response = await fetch(url);
      
      if (response.ok) {
        const data = await response.json();
        setAppeals(data || []);
      } else {
        setMessage({ type: 'error', text: 'Failed to fetch appeals' });
      }
    } catch (error) {
      console.error('Error fetching appeals:', error);
      setMessage({ type: 'error', text: 'Network error. Please try again.' });
    } finally {
      setLoading(false);
    }
  };

  const fetchTeacherAllocation = async (facultyId) => {
    try {
      console.log('Fetching allocation for faculty_id:', facultyId);
      const url = `${API_BASE_URL}/teachers/allocation?faculty_id=${facultyId}`;
      console.log('Request URL:', url);
      
      const response = await fetch(url);
      console.log('Response status:', response.status);
      
      if (response.ok) {
        const data = await response.json();
        console.log('Allocation data received:', data);
        setCurrentAllocations(data || []);
        setNewAllocations(data.map(a => ({ ...a })) || []);
      } else {
        const errorText = await response.text();
        console.error('Failed to fetch allocation:', response.status, errorText);
        setCurrentAllocations([]);
        setNewAllocations([]);
      }
    } catch (error) {
      console.error('Error fetching teacher allocation:', error);
      setCurrentAllocations([]);
      setNewAllocations([]);
    }
  };

  const handleOpenAcceptModal = async (appeal) => {
    console.log('Opening accept modal for appeal:', appeal);
    setSelectedAppeal(appeal);
    setHrMessage('');
    await fetchTeacherAllocation(appeal.faculty_id);
    setShowAcceptModal(true);
  };

  const handleReject = async (appeal) => {
    if (!window.confirm(`Are you sure you want to reject ${appeal.teacher_name}'s appeal?`)) {
      return;
    }

    setIsSubmitting(true);
    try {
      const response = await fetch(
        `${API_BASE_URL}/hr/appeals/${appeal.id}/resolve`,
        {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            hr_action: 'REJECTED',
            hr_message: '',
            new_counts: []
          })
        }
      );

      if (response.ok) {
        setMessage({ type: 'success', text: 'Appeal rejected successfully' });
        fetchAppeals();
        setTimeout(() => setMessage({ type: '', text: '' }), 3000);
      } else {
        const errorText = await response.text();
        setMessage({ type: 'error', text: errorText || 'Failed to reject appeal' });
      }
    } catch (error) {
      console.error('Error rejecting appeal:', error);
      setMessage({ type: 'error', text: 'Network error. Please try again.' });
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleAccept = async () => {
    setIsSubmitting(true);
    try {
      const requestBody = {
        hr_action: 'APPROVED',
        hr_message: hrMessage || '',
        new_counts: newAllocations.map(a => ({
          course_type_id: a.course_type_id,
          max_count: a.max_count
        }))
      };

      const response = await fetch(
        `${API_BASE_URL}/hr/appeals/${selectedAppeal.id}/resolve`,
        {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(requestBody)
        }
      );

      if (response.ok) {
        setMessage({ type: 'success', text: 'Appeal approved and allocations updated successfully' });
        setShowAcceptModal(false);
        setSelectedAppeal(null);
        fetchAppeals();
        setTimeout(() => setMessage({ type: '', text: '' }), 3000);
      } else {
        const errorText = await response.text();
        setMessage({ type: 'error', text: errorText || 'Failed to approve appeal' });
      }
    } catch (error) {
      console.error('Error approving appeal:', error);
      setMessage({ type: 'error', text: 'Network error. Please try again.' });
    } finally {
      setIsSubmitting(false);
    }
  };

  const updateAllocationCount = (courseTypeId, newCount) => {
    setNewAllocations(prev =>
      prev.map(a =>
        a.course_type_id === courseTypeId
          ? { ...a, max_count: Math.max(0, parseInt(newCount) || 0) }
          : a
      )
    );
  };

  const getDaysRemaining = () => {
    if (!windowStatus.endDate) return null;
    const now = new Date();
    const end = new Date(windowStatus.endDate);
    const diff = Math.ceil((end - now) / (1000 * 60 * 60 * 24));
    return diff;
  };

  return (
    <MainLayout 
    title="Teacher Workload Appeals">
      <div className="card-custom mx-auto py-6 px-4">
        {/* Header */}
        <div className="mb-6">
          <h1 className="text-3xl font-bold text-gray-900 mb-2">Teacher Workload Appeals</h1>
          <p className="text-gray-600">Review and process teacher course allocation appeal requests</p>
        </div>

        {/* Window Period Alert */}
        {windowStatus.isOpen && (
          <div className="mb-6 p-4 bg-orange-50 border-l-4 border-orange-500 rounded-r-lg">
            <div className="flex items-start">
              <svg className="w-6 h-6 text-orange-500 mr-3 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <div>
                <h3 className="text-lg font-semibold text-orange-900 mb-1">
                  ⏰ Course Selection Window is OPEN - Act Fast!
                </h3>
                <p className="text-orange-800 text-sm mb-2">
                  Teachers are waiting for your decision to select their courses for {windowStatus.academicYear}.
                </p>
                <p className="text-orange-900 font-medium">
                  Window closes in {getDaysRemaining()} day{getDaysRemaining() !== 1 ? 's' : ''}: {new Date(windowStatus.endDate).toLocaleDateString()}
                </p>
              </div>
            </div>
          </div>
        )}

        {/* Message */}
        {message.text && (
          <div
            className={`mb-4 p-3 rounded-lg text-base font-medium ${
              message.type === 'success'
                ? 'bg-green-50 text-primary border border-green-200'
                : 'bg-red-50 text-red-800 border border-red-200'
            }`}
          >
            {message.text}
          </div>
        )}

        {/* Filter Tabs */}
        <div className="mb-6 flex gap-2">
          <button
            onClick={() => setStatusFilter('pending')}
            className={`px-6 py-2 rounded-lg font-medium transition ${
              statusFilter === 'pending'
                ? 'bg-primary text-white shadow-lg'
                : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
            }`}
          >
            Pending ({appeals.filter(a => !a.appeal_status).length})
          </button>
          <button
            onClick={() => setStatusFilter('resolved')}
            className={`px-6 py-2 rounded-lg font-medium transition ${
              statusFilter === 'resolved'
                ? 'bg-primary text-white shadow-lg'
                : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
            }`}
          >
            Resolved
          </button>
          <button
            onClick={() => setStatusFilter('all')}
            className={`px-6 py-2 rounded-lg font-medium transition ${
              statusFilter === 'all'
                ? 'bg-primary text-white shadow-lg'
                : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
            }`}
          >
            All Appeals
          </button>
        </div>

        {/* Appeals Table */}
        {loading ? (
          <div className="text-center py-20">
            <p className="text-xl text-gray-600">Loading appeals...</p>
          </div>
        ) : appeals.length === 0 ? (
          <div className="text-center py-20 bg-white rounded-lg ">
            <p className="text-xl text-gray-600">No appeals found</p>
          </div>
        ) : (
          <div className="bg-white rounded-lg overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="bg-white border-b-2 border-gray-200">
                  <tr>
                    <th className="text-left py-3 px-4 font-semibold text-gray-900">Teacher</th>
                    <th className="text-left py-3 px-4 font-semibold text-gray-900">Appeal Message</th>
                    <th className="text-left py-3 px-4 font-semibold text-gray-900">Status</th>
                    <th className="text-left py-3 px-4 font-semibold text-gray-900">Date</th>
                    <th className="text-left py-3 px-4 font-semibold text-gray-900">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {appeals.map((appeal) => (
                    <tr key={appeal.id} className="border-b border-gray-100 hover:bg-gray-50">
                      <td className="py-3 px-4">
                        <p className="font-medium text-gray-900">{appeal.teacher_name}</p>
                        <p className="text-xs text-indigo-600 font-semibold">{appeal.teacher_faculty_id}</p>
                        <p className="text-xs text-gray-600">{appeal.teacher_email}</p>
                      </td>
                      <td className="py-3 px-4">
                        <p className="text-gray-700 max-w-md">{appeal.appeal_message}</p>
                      </td>
                      <td className="py-3 px-4">
                        {!appeal.appeal_status ? (
                          <span className="px-3 py-1 rounded-full text-xs font-medium bg-orange-100 text-orange-800">
                            Pending Review
                          </span>
                        ) : (
                          <span
                            className={`px-3 py-1 rounded-full text-xs font-medium ${
                              appeal.hr_action?.String === 'REJECTED'
                                ? 'bg-red-100 text-red-800'
                                : 'bg-background text-primary'
                            }`}
                          >
                            {appeal.hr_action?.String}
                          </span>
                        )}
                      </td>
                      <td className="py-3 px-4">
                        <p className="text-xs text-gray-600">
                          {new Date(appeal.created_at).toLocaleDateString()}
                        </p>
                      </td>
                      <td className="py-3 px-4">
                        {!appeal.appeal_status ? (
                          <div className="flex gap-2">
                            <button
                              onClick={() => handleOpenAcceptModal(appeal)}
                              disabled={isSubmitting}
                              className="px-4 py-1 bg-primary text-white rounded text-xs font-medium  disabled:opacity-50"
                            >
                              Accept
                            </button>
                            <button
                              onClick={() => handleReject(appeal)}
                              disabled={isSubmitting}
                              className="px-4 py-1 bg-red-600 text-white rounded text-xs font-medium hover:bg-red-700 disabled:opacity-50"
                            >
                              Reject
                            </button>
                          </div>
                        ) : (
                          <span className="text-xs text-gray-500">-</span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>

      {/* Accept Modal */}
      {showAcceptModal && selectedAppeal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-lg shadow-xl max-w-3xl w-full p-6 max-h-[90vh] overflow-y-auto">
            <h2 className="text-2xl font-bold text-gray-900 mb-4">Accept Appeal & Adjust Workload</h2>

            <div className="space-y-4">
              {/* Teacher Info */}
              <div className="bg-background p-4 rounded-lg border border-blue-200">
                <p className="text-sm text-gray-600 mb-1">Teacher:</p>
                <p className="text-lg font-medium text-gray-900">{selectedAppeal.teacher_name}</p>
                <p className="text-sm text-indigo-600 font-semibold">Faculty ID: {selectedAppeal.teacher_faculty_id}</p>
                <p className="text-sm text-gray-600">{selectedAppeal.teacher_email}</p>
                <p className="text-sm text-gray-600 mt-2">Appeal Message:</p>
                <p className="text-base text-gray-800">{selectedAppeal.appeal_message}</p>
              </div>

              {/* Current vs New Allocations */}
              <div>
                <h3 className="font-semibold text-gray-900 mb-3">Adjust Course Allocations:</h3>
                <div className="bg-gray-50 p-4 rounded-lg border border-gray-200">
                  {currentAllocations.length === 0 ? (
                    <p className="text-gray-600 text-sm">No current allocations found for this teacher.</p>
                  ) : (
                    <div className="space-y-3">
                      {newAllocations.map((alloc, idx) => (
                        <div key={alloc.course_type_id} className="flex items-center justify-between">
                          <div className="flex-1">
                            <label className="text-sm font-medium text-gray-700">
                              {alloc.course_type_name}
                            </label>
                          </div>
                          <div className="flex items-center gap-3">
                            <div className="text-sm text-gray-600">
                              Previous: <span className="font-semibold">{currentAllocations[idx]?.max_count || 0}</span>
                            </div>
                            <div className="flex items-center gap-2">
                              <label className="text-sm text-gray-600">New:</label>
                              <input
                                type="number"
                                min="0"
                                value={alloc.max_count}
                                onChange={(e) => updateAllocationCount(alloc.course_type_id, e.target.value)}
                                className="w-20 px-3 py-1 border border-gray-300 rounded focus:ring-2 focus:ring-primary focus:border-transparent text-center"
                              />
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>

              {/* HR Message */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Message to Teacher (Optional)
                </label>
                <textarea
                  value={hrMessage}
                  onChange={(e) => setHrMessage(e.target.value)}
                  placeholder="Add a message explaining the decision..."
                  rows="3"
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-none"
                />
              </div>

              {/* Buttons */}
              <div className="flex gap-3 pt-4">
                <button
                  onClick={() => {
                    setShowAcceptModal(false);
                    setSelectedAppeal(null);
                  }}
                  className="flex-1 px-4 py-2 border border-gray-300 rounded-lg text-gray-700 font-medium hover:bg-gray-50"
                  disabled={isSubmitting}
                >
                  Cancel
                </button>
                <button
                  onClick={handleAccept}
                  className="flex-1 px-4 py-2 bg-primary text-white rounded-lg font-medium disabled:opacity-50"
                  disabled={isSubmitting}
                >
                  {isSubmitting ? 'Processing...' : 'Approve & Update Allocation'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </MainLayout>
  );
};

export default HRAppealsReviewPage;
