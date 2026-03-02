import React, { useState, useEffect } from 'react';
import MainLayout from '../../components/MainLayout';
import { API_BASE_URL } from '../../config';

const HRFacultyPage = () => {
  const [faculty, setFaculty] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [searchTerm, setSearchTerm] = useState('');
  const [departmentFilter, setDepartmentFilter] = useState('all');
  const [statusFilter, setStatusFilter] = useState('all');
  const [selectedFaculty, setSelectedFaculty] = useState(null);
  const [courseTypes, setCourseTypes] = useState([]);
  const [courseLimits, setCourseLimits] = useState([]);
  const [saving, setSaving] = useState(false);
  const [saveMessage, setSaveMessage] = useState({ type: '', text: '' });

  useEffect(() => {
    fetchFaculty();
    fetchCourseTypes();
  }, []);

  const fetchCourseTypes = async () => {
    try {
      const response = await fetch(
        `${API_BASE_URL}/course-types`
      );
      if (response.ok) {
        const data = await response.json();
        console.log('Fetched course types:', data);
        setCourseTypes(data || []);
      } else {
        const errorData = await response.text();
        console.error('Failed to fetch course types:', errorData);
      }
    } catch (err) {
      console.error('Error fetching course types:', err);
    }
  };

  const fetchFaculty = async () => {
    try {
      setLoading(true);
      const response = await fetch(
        `${API_BASE_URL}/hr/faculty`
      );

      if (!response.ok) {
        throw new Error('Failed to fetch faculty data');
      }

      const data = await response.json();
      setFaculty(data.faculty || []);
      setLoading(false);
    } catch (err) {
      console.error('Error fetching faculty:', err);
      setError(err.message);
      setLoading(false);
    }
  };

  const handleFacultyClick = (f) => {
    console.log('Faculty clicked:', f);
    console.log('Available course types:', courseTypes);
    setSelectedFaculty(f);
    
    // Initialize course limits based on available types and faculty data
    if (!courseTypes || courseTypes.length === 0) {
      console.warn('No course types available to initialize limits');
      setCourseLimits([]);
    } else {
      const initialLimits = courseTypes.map(ct => {
        const existing = (f.course_limits || []).find(l => l.course_type_id === ct.id);
        return {
          course_type_id: ct.id,
          type_name: ct.name,
          max_count: existing ? existing.max_count : 0
        };
      });
      console.log('Initialized course limits:', initialLimits);
      setCourseLimits(initialLimits);
    }
    setSaveMessage({ type: '', text: '' });
  };

  const handleLimitChange = (typeId, value) => {
    const num = value === '' ? 0 : parseInt(value, 10);
    if (!isNaN(num) && num >= 0) {
      setCourseLimits(prev => prev.map(limit => 
        limit.course_type_id === typeId ? { ...limit, max_count: num } : limit
      ));
    }
  };

  const handleSave = async () => {
    if (!selectedFaculty) return;

    try {
      setSaving(true);
      const response = await fetch(
        `${API_BASE_URL}/hr/faculty/subject-counts`,
        {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            faculty_id: selectedFaculty.id,
            course_limits: courseLimits
          })
        }
      );

      if (response.ok) {
        setFaculty(faculty.map(f => 
          f.id === selectedFaculty.id 
            ? { ...f, course_limits: courseLimits }
            : f
        ));
        setSelectedFaculty({ 
          ...selectedFaculty, 
          course_limits: courseLimits 
        });
        setSaveMessage({ type: 'success', text: 'Subject counts updated successfully!' });
        setTimeout(() => setSaveMessage({ type: '', text: '' }), 3000);
      } else {
        const error = await response.json();
        setSaveMessage({ type: 'error', text: error.error || 'Failed to save' });
      }
    } catch (err) {
      console.error('Error saving:', err);
      setSaveMessage({ type: 'error', text: 'Network error. Please try again.' });
    } finally {
      setSaving(false);
    }
  };

  const departments = [...new Set(faculty.map(f => f.department_name?.String).filter(Boolean))];

  const filteredFaculty = faculty.filter(f => {
    const matchesSearch = 
      f.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      f.faculty_id.toLowerCase().includes(searchTerm.toLowerCase()) ||
      f.email.toLowerCase().includes(searchTerm.toLowerCase());

    const matchesDepartment = 
      departmentFilter === 'all' || 
      f.department_name?.String === departmentFilter;

    const matchesStatus = 
      statusFilter === 'all' || 
      (statusFilter === 'active' && f.status) ||
      (statusFilter === 'inactive' && !f.status);

    return matchesSearch && matchesDepartment && matchesStatus;
  });

  if (loading) {
    return (
      <MainLayout>
        <div className="flex items-center justify-center min-h-screen">
          <div className="text-2xl text-gray-700">Loading faculty data...</div>
        </div>
      </MainLayout>
    );
  }

  if (error) {
    return (
      <MainLayout>
        <div className="max-w-7xl mx-auto py-12">
          <div className="text-center">
            <h1 className="text-3xl font-bold text-red-600 mb-4">Error</h1>
            <p className="text-xl text-gray-600">{error}</p>
          </div>
        </div>
      </MainLayout>
    );
  }

  return (
    <MainLayout title="Faculty Directory">
      <div className="h-screen flex flex-col overflow-hidden bg-gray-50">
        {/* Header */}
        <div className="flex-none bg-white border-b border-gray-200 px-12 py-8">
          <div className="max-w-7xl mx-auto flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold text-gray-900 mb-2">Faculty Directory</h1>
              <p className="text-base text-gray-600">Manage faculty and subject allocations</p>
            </div>
            <div className="text-right">
              <div className="text-3xl font-bold text-primary">{filteredFaculty.length}</div>
              <div className="text-xs text-gray-500 uppercase tracking-wide mt-1">Total Faculty</div>
            </div>
          </div>
        </div>

        {/* Filters */}
        <div className="flex-none bg-white border-b border-gray-200 px-12 py-6">
          <div className="max-w-7xl mx-auto grid grid-cols-3 gap-8">
            <div>
              <label className="block text-xs font-semibold text-gray-700 mb-3 uppercase tracking-wide">
                Search Faculty
              </label>
              <input
                type="text"
                placeholder="Name, ID, or email..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="w-full px-5 py-3.5 text-base border border-gray-300 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
              />
            </div>
            <div>
              <label className="block text-xs font-semibold text-gray-700 mb-3 uppercase tracking-wide">
                Filter by Department
              </label>
              <select
                value={departmentFilter}
                onChange={(e) => setDepartmentFilter(e.target.value)}
                className="w-full px-5 py-3.5 text-base border border-gray-300 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
              >
                <option value="all">All Departments</option>
                {departments.map(dept => (
                  <option key={dept} value={dept}>{dept}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-xs font-semibold text-gray-700 mb-3 uppercase tracking-wide">
                Filter by Status
              </label>
              <select
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value)}
                className="w-full px-5 py-3.5 text-base border border-gray-300 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
              >
                <option value="all">All Status</option>
                <option value="active">Active</option>
                <option value="inactive">Inactive</option>
              </select>
            </div>
          </div>
        </div>

        {/* Main Content */}
        <div className="flex-1 flex overflow-hidden relative">
          {/* Faculty List */}
          <div className="w-full bg-white flex flex-col transition-all duration-300">
            <div className="flex-1 overflow-y-auto">
              <div className="max-w-6xl mx-auto px-12 py-8">
                <table className="min-w-full">
                  <thead className="sticky top-0 bg-gray-50 z-10">
                    <tr className="border-b-2 border-gray-200">
                      <th className="px-6 py-5 text-left text-xs font-bold text-gray-600 uppercase tracking-wider">
                        Faculty ID
                      </th>
                      <th className="px-6 py-5 text-left text-xs font-bold text-gray-600 uppercase tracking-wider">
                        Name
                      </th>
                      <th className="px-6 py-5 text-left text-xs font-bold text-gray-600 uppercase tracking-wider">
                        Email
                      </th>
                      <th className="px-6 py-5 text-left text-xs font-bold text-gray-600 uppercase tracking-wider">
                        Status
                      </th>
                    </tr>
                  </thead>
                  <tbody className="bg-white">
                    {filteredFaculty.length === 0 ? (
                      <tr>
                        <td colSpan="4" className="px-6 py-20 text-center text-gray-500">
                          <div className="flex flex-col items-center">
                            <svg className="w-16 h-16 text-gray-300 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
                            </svg>
                            <p className="text-lg font-semibold text-gray-700 mb-1">No faculty members found</p>
                            <p className="text-sm text-gray-500">Try adjusting your filters</p>
                          </div>
                        </td>
                      </tr>
                    ) : (
                      filteredFaculty.map((f) => (
                        <tr 
                          key={f.id} 
                          onClick={() => handleFacultyClick(f)}
                          className={`cursor-pointer hover:bg-blue-50 transition-all border-b border-gray-100 ${
                            selectedFaculty?.id === f.id ? 'bg-blue-50 shadow-inner' : ''
                          }`}
                        >
                          <td className="px-6 py-6 whitespace-nowrap">
                            <span className="text-sm font-bold text-gray-900">{f.faculty_id}</span>
                          </td>
                          <td className="px-6 py-6 whitespace-nowrap">
                            <div className="flex items-center">
                              {f.profile_img?.Valid && f.profile_img.String ? (
                                <img
                                  src={f.profile_img.String}
                                  alt={f.name}
                                  className="h-12 w-12 rounded-full mr-4 object-cover border-2 border-gray-200 shadow"
                                />
                              ) : (
                                <div className="h-12 w-12 rounded-full bg-gradient-to-br from-blue-400 to-blue-600 flex items-center justify-center mr-4 shadow">
                                  <span className="text-white font-bold text-base">
                                    {f.name.charAt(0).toUpperCase()}
                                  </span>
                                </div>
                              )}
                              <span className="text-base font-semibold text-gray-900">{f.name}</span>
                            </div>
                          </td>
                          <td className="px-6 py-6 whitespace-nowrap">
                            <span className="text-sm text-gray-600">{f.email}</span>
                          </td>
                          <td className="px-6 py-6 whitespace-nowrap">
                            <span className={`px-4 py-2 inline-flex text-xs leading-5 font-bold rounded-full ${
                              f.status 
                                ? 'bg-green-100 text-green-700' 
                                : 'bg-red-100 text-red-700'
                            }`}>
                              {f.status ? 'Active' : 'Inactive'}
                            </span>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          </div>

          {/* Faculty details modal popup */}
          {selectedFaculty && (
            <div
              className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50"
              onClick={() => setSelectedFaculty(null)}
            >
              <div
                className="bg-white rounded-2xl shadow-2xl max-w-md w-full max-h-[90vh] overflow-auto"
                onClick={(e) => e.stopPropagation()}
              >
                <div className="flex-none bg-white border-b border-gray-200 px-6 py-4 flex items-center justify-between">
                  <h2 className="text-lg font-bold text-gray-900">Faculty Details</h2>
                  <button
                    onClick={() => setSelectedFaculty(null)}
                    className="p-2 rounded-lg text-gray-500 hover:text-gray-700 hover:bg-gray-100 transition-colors"
                    aria-label="Close faculty details"
                  >
                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                </div>

                <div className="p-6">
                  {/* Compact Faculty Info */}
                  <div className="mb-6">
                    <div className="flex items-start mb-4">
                      {selectedFaculty.profile_img?.Valid && selectedFaculty.profile_img.String ? (
                        <img
                          src={selectedFaculty.profile_img.String}
                          alt={selectedFaculty.name}
                          className="h-16 w-16 rounded-lg mr-4 object-cover border-2 border-gray-100"
                        />
                      ) : (
                        <div className="h-16 w-16 rounded-lg bg-gradient-to-br from-blue-400 to-blue-600 flex items-center justify-center mr-4">
                          <span className="text-white font-bold text-xl">
                            {selectedFaculty.name.charAt(0).toUpperCase()}
                          </span>
                        </div>
                      )}
                      <div className="flex-1">
                        <div className="text-base font-semibold text-gray-900">{selectedFaculty.name}</div>
                        <div className="text-xs text-gray-500 mt-1">{selectedFaculty.faculty_id}</div>
                        <div className="text-xs text-gray-600 mt-1">
                          {selectedFaculty.department_name?.Valid ? selectedFaculty.department_name.String : '-'}
                        </div>
                        <div className="text-xs text-gray-500 mt-1">
                          {selectedFaculty.desg?.Valid ? selectedFaculty.desg.String : '-'}
                        </div>
                      </div>
                    </div>

                    <div className="space-y-3 text-sm">
                      <div>
                        <span className="font-medium text-gray-600">Email: </span>
                        <span className="text-gray-900">{selectedFaculty.email}</span>
                      </div>
                      {selectedFaculty.phone?.Valid && (
                        <div>
                          <span className="font-medium text-gray-600">Phone: </span>
                          <span className="text-gray-900">{selectedFaculty.phone.String}</span>
                        </div>
                      )}
                    </div>
                  </div>

                  {/* Subject counts inputs dynamic */}
                  <div className="space-y-4">
                    {courseLimits.map((limit) => (
                      <div key={limit.course_type_id}>
                        <label className="block text-sm font-medium text-gray-700 mb-2 capitalize">
                          {limit.type_name} Subject Count
                        </label>
                        <input
                          type="number"
                          min="0"
                          value={limit.max_count}
                          onChange={(e) => handleLimitChange(limit.course_type_id, e.target.value)}
                          className="w-full px-4 py-2 text-sm border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 transition-all"
                        />
                      </div>
                    ))}

                    {saveMessage.text && (
                      <div className={`p-3 rounded-lg ${saveMessage.type === 'success' ? 'bg-green-50 text-green-800' : 'bg-red-50 text-red-800'}`}>
                        <div className="text-sm font-semibold">{saveMessage.text}</div>
                      </div>
                    )}

                    <button
                      onClick={handleSave}
                      disabled={saving}
                      className="w-full bg-blue-600 text-white py-2.5 rounded-lg text-sm font-semibold hover:bg-blue-700 disabled:opacity-60 transition-colors"
                    >
                      {saving ? 'Saving...' : 'Save Subject Counts'}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </MainLayout>
  );
};

export default HRFacultyPage;
