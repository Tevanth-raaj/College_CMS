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
  const [importing, setImporting] = useState(false);
  const [importMessage, setImportMessage] = useState({ type: '', text: '' });

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
            faculty_id: selectedFaculty.faculty_id,
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

  const handleExportTemplate = () => {
    try {
      const headers = ['faculty_id', 'name', 'email', 'department', ...courseTypes.map(ct => `${ct.name} (${ct.id})`)];
      const rows = faculty.map(f => [
        f.faculty_id,
        f.name,
        f.email,
        f.department_name?.String || '',
        ...courseTypes.map(ct => {
          const existing = (f.course_limits || []).find(l => l.course_type_id === ct.id);
          return existing ? existing.max_count : 0;
        })
      ]);

      // Create CSV content
      let csvContent = headers.join(',') + '\n';
      rows.forEach(row => {
        csvContent += row.map(cell => `"${cell}"`).join(',') + '\n';
      });

      // Download CSV file
      const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
      const link = document.createElement('a');
      const url = URL.createObjectURL(blob);
      link.setAttribute('href', url);
      link.setAttribute('download', 'faculty_workload_template.csv');
      link.style.visibility = 'hidden';
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
    } catch (err) {
      console.error('Error exporting template:', err);
      setImportMessage({ type: 'error', text: 'Failed to export template' });
    }
  };

  const handleDownloadTemplate = () => {
    try {
      const headers = ['faculty_id', 'name', 'email', 'department', ...courseTypes.map(ct => `${ct.name} (${ct.id})`)];
      const exampleRow = ['F001', 'Dr. John Smith', 'john@example.com', 'Computer Science', ...courseTypes.map(() => 3)];

      let csvContent = '# Instructions: Fill in the course counts for each faculty member. Do not modify faculty_id, name, email, or department.\n';
      csvContent += headers.join(',') + '\n';
      csvContent += exampleRow.map(cell => `"${cell}"`).join(',') + '\n';

      // Download CSV file
      const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
      const link = document.createElement('a');
      const url = URL.createObjectURL(blob);
      link.setAttribute('href', url);
      link.setAttribute('download', 'faculty_workload_template_blank.csv');
      link.style.visibility = 'hidden';
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
    } catch (err) {
      console.error('Error downloading template:', err);
      setImportMessage({ type: 'error', text: 'Failed to download template' });
    }
  };

  const handleImportTemplate = async (file) => {
    if (!file) return;

    try {
      setImporting(true);
      setImportMessage({ type: '', text: '' });

      const text = await file.text();
      const lines = text.split('\n').filter(line => line.trim() && !line.startsWith('#'));
      const headers = lines[0].split(',').map(h => h.trim().replace(/^"|"$/g, ''));
      
      const facultyIdIndex = headers.findIndex(h => h.toLowerCase() === 'faculty_id');
      if (facultyIdIndex === -1) {
        throw new Error('CSV must contain faculty_id column');
      }

      let successCount = 0;
      let failureCount = 0;
      const errors = [];

      for (let i = 1; i < lines.length; i++) {
        const values = lines[i].split(',').map(v => v.trim().replace(/^"|"$/g, ''));
        const facultyId = values[facultyIdIndex];

        if (!facultyId) continue;

        // Find the faculty member
        const f = faculty.find(fac => fac.faculty_id === facultyId);
        if (!f) {
          errors.push(`Row ${i + 1}: Faculty ID ${facultyId} not found`);
          failureCount++;
          continue;
        }

        // Extract course limits from columns starting after department
        const courseStartIndex = headers.findIndex(h => h.toLowerCase() === 'department') + 1;
        const limits = courseTypes.map((ct, idx) => ({
          course_type_id: ct.id,
          type_name: ct.name,
          max_count: parseInt(values[courseStartIndex + idx]) || 0
        }));

        // Save these limits
        try {
          const response = await fetch(
            `${process.env.REACT_APP_API_BASE_URL || 'http://localhost:5000/api'}/hr/faculty/subject-counts`,
            {
              method: 'PUT',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({
                faculty_id: f.faculty_id,
                course_limits: limits
              })
            }
          );

          if (response.ok) {
            setFaculty(faculty.map(fac => 
              fac.id === f.id 
                ? { ...fac, course_limits: limits }
                : fac
            ));
            successCount++;
          } else {
            errors.push(`Row ${i + 1}: Failed to save for ${facultyId}`);
            failureCount++;
          }
        } catch (err) {
          errors.push(`Row ${i + 1}: ${err.message}`);
          failureCount++;
        }
      }

      const message = `Import completed: ${successCount} successful, ${failureCount} failed`;
      if (errors.length > 0 && errors.length <= 5) {
        setImportMessage({ type: failureCount === 0 ? 'success' : 'warning', text: message + '\n' + errors.join('\n') });
      } else if (errors.length > 5) {
        setImportMessage({ type: failureCount === 0 ? 'success' : 'warning', text: message + ` (${errors.length} errors)` });
      } else {
        setImportMessage({ type: 'success', text: message });
      }

      setTimeout(() => setImportMessage({ type: '', text: '' }), 5000);
    } catch (err) {
      console.error('Error importing:', err);
      setImportMessage({ type: 'error', text: err.message });
    } finally {
      setImporting(false);
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
    <MainLayout
    >
      <div className="h-screen flex flex-col overflow-hidden bg-gradient-to-br from-gray-50 to-gray-100">
        {/* Header with Actions */}
        <div className="flex-none bg-white border-b border-gray-200 px-8 py-6 shadow-sm">
          <div className="max-w-7xl mx-auto flex items-center justify-between mb-6">
            <div>
              <h1 className="text-4xl font-bold text-gray-900">Faculty Workload Management</h1>
              <p className="text-base text-gray-600 mt-2">Assign and manage course requirements for faculty members</p>
            </div>
            <div className="text-right bg-gradient-to-br from-blue-50 to-primary_dim px-6 py-4 rounded-xl border border-primary">
              <div className="text-3xl font-bold text-primary">{filteredFaculty.length}</div>
              <div className="text-xs text-primary uppercase tracking-wide font-semibold mt-1">Faculties Listed</div>
            </div>
          </div>

          {/* Action Buttons */}
          <div className="flex gap-3 items-center">
            <button
              onClick={handleDownloadTemplate}
              className="inline-flex items-center gap-2 px-4 py-2.5 bg-primary text-white rounded-lg font-semibold hover:shadow-lg hover:from-blue-600 hover:to-blue-700 transition-all"
              title="Download blank template for bulk import"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
              </svg>
              Download Template
            </button>

            <button
              onClick={handleExportTemplate}
              className="inline-flex items-center gap-2 px-4 py-2.5 bg-primary text-white rounded-lg font-semibold hover:shadow-lg hover:from-emerald-600 hover:to-emerald-700 transition-all"
              title="Export current workload assignments"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2m0 0v-8m0 8l-6-4m6 4l6-4" />
              </svg>
              Export Current
            </button>

            <label className="inline-flex items-center gap-2 px-4 py-2.5 bg-primary text-white rounded-lg font-semibold hover:shadow-lg hover:from-purple-600 hover:to-purple-700 transition-all cursor-pointer">
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
              </svg>
              Import Workload
              <input
                type="file"
                accept=".csv"
                onChange={(e) => {
                  if (e.target.files?.[0]) {
                    handleImportTemplate(e.target.files[0]);
                  }
                }}
                disabled={importing}
                className="hidden"
              />
            </label>
          </div>

          {/* Import Message */}
          {importMessage.text && (
            <div className={`mt-4 p-4 rounded-xl text-sm font-medium whitespace-pre-wrap ${
              importMessage.type === 'success' 
                ? 'bg-green-50 text-green-800 border border-green-200' 
                : importMessage.type === 'warning'
                ? 'bg-yellow-50 text-yellow-800 border border-yellow-200'
                : 'bg-red-50 text-red-800 border border-red-200'
            }`}>
              {importMessage.text}
            </div>
          )}
        </div>

        {/* Filters */}
        <div className="flex-none bg-white border-b border-gray-100 px-8 py-5">
          <div className="max-w-7xl mx-auto grid grid-cols-3 gap-6">
            <div>
              <label className="block text-xs font-bold text-gray-600 mb-2 uppercase tracking-widest">
                Search Faculty
              </label>
              <input
                type="text"
                placeholder="Name, ID, or email..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="w-full px-4 py-3 text-sm border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
              />
            </div>
            <div>
              <label className="block text-xs font-bold text-gray-600 mb-2 uppercase tracking-widest">
                Department
              </label>
              <select
                value={departmentFilter}
                onChange={(e) => setDepartmentFilter(e.target.value)}
                className="w-full px-4 py-3 text-sm border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
              >
                <option value="all">All Departments</option>
                {departments.map(dept => (
                  <option key={dept} value={dept}>{dept}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-xs font-bold text-gray-600 mb-2 uppercase tracking-widest">
                Status
              </label>
              <select
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value)}
                className="w-full px-4 py-3 text-sm border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
              >
                <option value="all">All Status</option>
                <option value="active">Active</option>
                <option value="inactive">Inactive</option>
              </select>
            </div>
          </div>
        </div>

        {/* Main Content */}
        <div className="flex-1 overflow-hidden bg-gray-50">
          {/* Faculty List */}
          <div className="w-full h-full flex flex-col">
            <div className="flex-1 px-8 py-8 overflow-hidden">
              <div className="max-w-7xl mx-auto h-full">
                <div className="bg-white rounded-xl border border-gray-200 shadow-sm overflow-auto h-full">
                  <table className="min-w-full relative">
                    <thead className="sticky top-0 bg-gradient-to-r from-gray-50 to-gray-100 z-10 shadow-sm">
                      <tr className="border-b-2 border-gray-200">
                        <th className="px-6 py-5 text-left text-xs font-bold text-gray-700 uppercase tracking-wider bg-gradient-to-r from-gray-50 to-gray-100">
                          Faculty ID
                        </th>
                        <th className="px-6 py-5 text-left text-xs font-bold text-gray-700 uppercase tracking-wider bg-gradient-to-r from-gray-50 to-gray-100">
                          Name
                        </th>
                        <th className="px-6 py-5 text-left text-xs font-bold text-gray-700 uppercase tracking-wider bg-gradient-to-r from-gray-50 to-gray-100">
                          Email
                        </th>
                        <th className="px-6 py-5 text-left text-xs font-bold text-gray-700 uppercase tracking-wider bg-gradient-to-r from-gray-50 to-gray-100">
                          Status
                        </th>
                        {courseTypes.map((courseType, idx) => {
                          // Get proper abbreviation for course type
                          const getAbbreviation = (name) => {
                            const lowerName = name.toLowerCase();
                            // Check for combined types first
                            if (lowerName.includes('theory') && lowerName.includes('lab')) return 'TL';
                            if (lowerName.includes('tutorial')) return 'TL';
                            if (lowerName.includes('theory')) return 'T';
                            if (lowerName.includes('lab')) return 'L';
                            // Fallback to first letter if no match
                            return name.substring(0, 1).toUpperCase();
                          };
                          
                          return (
                            <th key={courseType.id} className="px-4 py-5 text-center text-xs font-bold text-gray-700 uppercase tracking-wider bg-gradient-to-r from-gray-50 to-gray-100">
                              <div className="flex flex-col items-center">
                                <span className="mb-1">{getAbbreviation(courseType.name)}</span>
                              </div>
                            </th>
                          );
                        })}
                      </tr>
                    </thead>
                    <tbody className="bg-white">
                      {filteredFaculty.length === 0 ? (
                        <tr>
                          <td colSpan={4 + courseTypes.length} className="px-6 py-20 text-center text-gray-500">
                            <div className="flex flex-col items-center">
                              <svg className="w-16 h-16 text-gray-300 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
                              </svg>
                              <p className="text-lg font-bold text-gray-600 mb-1">No faculty members found</p>
                              <p className="text-sm text-gray-500">Try adjusting your filters or search criteria</p>
                            </div>
                          </td>
                        </tr>
                      ) : (
                        filteredFaculty.map((f) => {
                          // Create a map for quick lookup
                          const limitsMap = {};
                          (f.course_limits || []).forEach(limit => {
                            limitsMap[limit.course_type_id] = limit.max_count;
                          });
                          
                          return (
                            <tr 
                              key={f.id} 
                              onClick={() => handleFacultyClick(f)}
                              className="cursor-pointer hover:bg-blue-50 transition-all border-b border-gray-100"
                            >
                              <td className="px-6 py-5 whitespace-nowrap">
                                <span className="text-sm font-bold text-gray-900 bg-gray-100 px-3 py-1 rounded">{f.faculty_id}</span>
                              </td>
                              <td className="px-6 py-5 whitespace-nowrap">
                                <div className="flex items-center">
                                  {f.profile_img?.Valid && f.profile_img.String ? (
                                    <img
                                      src={f.profile_img.String}
                                      alt={f.name}
                                      className="h-10 w-10 rounded-full mr-3 object-cover border-2 border-gray-200 shadow"
                                    />
                                  ) : (
                                    <div className="h-10 w-10 rounded-full bg-gradient-to-br from-primary to-primary_medium flex items-center justify-center mr-3 shadow text-white font-bold text-sm">
                                      {f.name.charAt(0).toUpperCase()}
                                    </div>
                                  )}
                                  <span className="text-base font-semibold text-gray-900">{f.name}</span>
                                </div>
                              </td>
                              <td className="px-6 py-5 whitespace-nowrap">
                                <span className="text-sm text-gray-600">{f.email}</span>
                              </td>
                              <td className="px-6 py-5 whitespace-nowrap">
                                <span className={`px-3 py-1.5 inline-flex text-xs leading-5 font-bold rounded-full ${
                                  f.status 
                                    ? 'bg-green-100 text-green-700' 
                                    : 'bg-red-100 text-red-700'
                                }`}>
                                  {f.status ? 'Active' : 'Inactive'}
                                </span>
                              </td>
                              {courseTypes.map((courseType) => (
                                <td key={courseType.id} className="px-4 py-5 whitespace-nowrap text-center">
                                  <span className="text-sm font-semibold text-primary">
                                    {limitsMap[courseType.id] || 0}
                                  </span>
                                </td>
                              ))}
                            </tr>
                          );
                        })
                      )}
                    </tbody>
                  </table>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Modal Overlay */}
        {selectedFaculty && (
          <div className="fixed inset-0 bg-black bg-opacity-50 z-40 flex items-center justify-center p-4">
            {/* Modal */}
            <div className="bg-white rounded-2xl shadow-2xl w-full max-w-md max-h-[90vh] overflow-y-auto animate-in fade-in zoom-in-95 duration-300">
              {/* Modal Header */}
              <div className="sticky top-0 bg-primary text-white px-8 py-6 flex items-center justify-between">
                <div>
                  <h2 className="text-2xl font-bold">Workload Assignment</h2>
                  <p className="text-indigo-100 text-sm mt-1">Configure course requirements</p>
                </div>
                <button
                  onClick={() => setSelectedFaculty(null)}
                  className="p-2 rounded-lg text-white transition-colors"
                  aria-label="Close modal"
                >
                  <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>

              {/* Modal Content */}
              <div className="p-8">
                {/* Faculty Info Card */}
                <div className="mb-8 p-6 bg-background rounded-xl border-2 border-indigo-200">
                  <div className="flex items-start mb-5">
                    {selectedFaculty.profile_img?.Valid && selectedFaculty.profile_img.String ? (
                      <img
                        src={selectedFaculty.profile_img.String}
                        alt={selectedFaculty.name}
                        className="h-20 w-20 rounded-lg mr-5 object-cover border-3 border-white shadow-lg"
                      />
                    ) : (
                      <div className="h-20 w-20 rounded-lg bg-gradient-to-br from-indigo-400 to-blue-600 flex items-center justify-center mr-5 text-white shadow-lg">
                        <span className="font-bold text-3xl">{selectedFaculty.name.charAt(0).toUpperCase()}</span>
                      </div>
                    )}
                    <div className="flex-1">
                      <div className="text-xl font-bold text-gray-900">{selectedFaculty.name}</div>
                      <div className="text-xs text-gray-600 mt-1 font-mono bg-white px-3 py-1 rounded inline-block border border-indigo-200">
                        ID: {selectedFaculty.faculty_id}
                      </div>
                      {selectedFaculty.department_name?.String && (
                        <div className="text-sm text-gray-700 mt-3 font-semibold"> {selectedFaculty.department_name.String}</div>
                      )}
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-3 text-xs">
                    <div className="p-3 bg-white rounded-lg border border-indigo-100">
                      <span className="text-gray-600 font-bold uppercase tracking-wider">Email</span>
                      <div className="text-gray-900 font-semibold truncate mt-1">{selectedFaculty.email}</div>
                    </div>
                    <div className="p-3 bg-white rounded-lg border border-indigo-100">
                      <span className="text-gray-600 font-bold uppercase tracking-wider">Status</span>
                      <div className="mt-1">
                        <span className={`px-3 py-1 text-xs font-bold rounded-full inline-block ${
                          selectedFaculty.status 
                            ? 'bg-green-100 text-green-700' 
                            : 'bg-red-100 text-red-700'
                        }`}>
                          {selectedFaculty.status ? '✓ Active' : '✗ Inactive'}
                        </span>
                      </div>
                    </div>
                  </div>
                </div>

                {/* Workload Assignment Section */}
                <div className="space-y-6">
                  <div>
                    <h2 className="text-lg font-bold text-gray-900 mb-6">Course Type Quotas</h2>

                    <div className="space-y-4">
                      {courseLimits.length === 0 ? (
                        <div className="p-5 bg-yellow-50 border-2 border-yellow-300 rounded-lg text-sm text-yellow-800 font-semibold">
                          ⚠️ No course types configured. Please check system settings.
                        </div>
                      ) : (
                        courseLimits.map((limit, idx) => (
                          <div key={limit.course_type_id} className="group">
                            <div className="flex items-center justify-between mb-3">
                              <label className="text-base font-bold text-gray-800 capitalize flex items-center gap-2">
                                <span className={`w-3 h-3 rounded-full ${idx === 0 ? 'bg-blue-500' : idx === 1 ? 'bg-purple-500' : 'bg-pink-500'}`}></span>
                                {limit.type_name}
                              </label>
                              <span className="text-sm font-semibold text-gray-700">
                                {limit.max_count}
                              </span>
                            </div>
                            <div className="relative">
                              <input
                                type="number"
                                min="0"
                                max="20"
                                value={limit.max_count}
                                onChange={(e) => handleLimitChange(limit.course_type_id, e.target.value)}
                                className="input-custom"
                              />
                              {/* removed decorative icon for a cleaner, professional look */}
                            </div>
                            <div className="mt-2 h-2 bg-gray-200 rounded-full overflow-hidden">
                              <div 
                                className="h-full bg-gradient-to-r from-indigo-500 to-blue-500 transition-all"
                                style={{ width: `${Math.min(limit.max_count * 5, 100)}%` }}
                              ></div>
                            </div>
                          </div>
                        ))
                      )}
                    </div>
                  </div>

                  {/* Messages */}
                  {saveMessage.text && (
                    <div className={`p-4 rounded-xl text-sm font-semibold border-l-4 ${
                      saveMessage.type === 'success' 
                        ? 'bg-green-50 text-green-800 border-green-500' 
                        : 'bg-red-50 text-red-800 border-red-500'
                    }`}>
                      <div className="flex items-start gap-3">
                        {saveMessage.type === 'success' ? (
                          <svg className="w-5 h-5 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                            <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                          </svg>
                        ) : (
                          <svg className="w-5 h-5 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                            <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
                          </svg>
                        )}
                        <span>{saveMessage.text}</span>
                      </div>
                    </div>
                  )}

                  {/* Action Buttons */}
                  <div className="space-y-3 pt-2">
                    <button
                      onClick={handleSave}
                      disabled={saving}
                      className="w-full bg-gradient-to-r from-indigo-600 to-blue-600 text-white py-4 rounded-xl text-base font-bold hover:shadow-xl hover:from-indigo-700 hover:to-blue-700 disabled:opacity-60 transition-all flex items-center justify-center gap-2"
                    >
                      {saving ? (
                        <>
                          <svg className="w-5 h-5 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                          </svg>
                          Saving...
                        </>
                      ) : (
                        <>
                          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                          </svg>
                          Save Workload Assignment
                        </>
                      )}
                    </button>

                    <button
                      onClick={() => setSelectedFaculty(null)}
                      className="w-full border-2 border-gray-300 text-gray-700 py-3 rounded-xl text-base font-bold hover:bg-gray-50 transition-all"
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </MainLayout>
  );
};

export default HRFacultyPage;
