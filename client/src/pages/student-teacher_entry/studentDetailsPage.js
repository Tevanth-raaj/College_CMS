  import React, { useState, useEffect } from 'react'
  import MainLayout from '../../components/MainLayout'
  import StudentCard from '../../components/StudentCard'
  import SearchBarWithDropdown from '../../components/SearchBarWithDropdown'
  import { API_BASE_URL } from '../../config'

  function 
  StudentDetailsPage() {
    const [curriculumDisplay, setCurriculumDisplay] = useState('')
    const [formData, setFormData] = useState({
      // Basic Student Fields
      enrollment_no: '',
      register_no: '',
      dte_reg_no: '',
      application_no: '',
      admission_no: '',
      student_name: '',
      gender: '',
      dob: '',
      age: '',
      father_name: '',
      mother_name: '',
      guardian_name: '',
      religion: '',
      nationality: 'Indian',
      community: '',
      mother_tongue: '',
      blood_group: '',
      aadhar_no: '',
      parent_occupation: '',
      designation: '',
      place_of_work: '',
      parent_income: '',
      
      // Academic Details Fields
      batch: '',
      year: '',
      semester: '',
      degree_level: '',
      section: '',
      department: '',
      student_category: '',
      branch_type: '',
      seat_category: '',
      regulation: '',
      quota: '',
      university: '',
      year_of_admission: '',
      year_of_completion: '',
      student_status: '',
      curriculum_id: '',
      
      // Address Fields
      permanent_address: '',
      present_address: '',
      residence_location: '',
      
      // Admission Payment Fields
      dte_register_no: '',
      dte_admission_no: '',
      receipt_no: '',
      receipt_date: '',
      amount: '',
      bank_name: '',
      
      // Contact Details Fields
      parent_mobile: '',
      student_mobile: '',
      student_email: '',
      parent_email: '',
      official_email: '',
      
      // Hostel Details Fields
      hosteller_type: '',
      hostel_name: '',
      room_no: '',
      room_capacity: '',
      room_type: '',
      floor_no: '',
      warden_name: '',
      alternate_warden: '',
      class_advisor: '',
      
      // Insurance Details Fields
      nominee_name: '',
      relationship: '',
      nominee_age: '',
      
      // School Details - Array
      school_details: [
        { school_name: '', board: '', year_of_pass: '', state: '', tc_no: '', tc_date: '', total_marks: '' }
      ]
    })

    const [error, setError] = useState('')
    const [success, setSuccess] = useState('')
    const [loading, setLoading] = useState(false)
    const [students, setStudents] = useState([])
    const [curriculums, setCurriculums] = useState([])
    const [departments, setDepartments] = useState([])
    const [searchTerm, setSearchTerm] = useState('')
    const [showForm, setShowForm] = useState(false)
    const [editingStudent, setEditingStudent] = useState(null)

    // Auto-calculate age from DOB
    const calculateAge = (dob) => {
      if (!dob) return ''
      const today = new Date()
      const birthDate = new Date(dob)
      let age = today.getFullYear() - birthDate.getFullYear()
      const monthDiff = today.getMonth() - birthDate.getMonth()
      if (monthDiff < 0 || (monthDiff === 0 && today.getDate() < birthDate.getDate())) {
        age--
      }
      return age.toString()
    }

    const handleInputChange = (e) => {
      const { name, value } = e.target
      
      // Auto-calculate age when DOB changes
      if (name === 'dob') {
        const calculatedAge = calculateAge(value)
        setFormData(prev => ({
          ...prev,
          [name]: value,
          age: calculatedAge
        }))
      } else {
        setFormData(prev => ({
          ...prev,
          [name]: value
        }))
      }
    }

    // Convert form data to proper types for backend (all strings)
    const formatFormDataForSubmission = (data) => {
      const formatted = { ...data }
      
      // Handle date fields - set to null if empty string to avoid DB errors
      const dateFields = ['dob', 'receipt_date']
      dateFields.forEach(field => {
        if (formatted[field] === '') {
          formatted[field] = null
        }
      })

      // Ensure all numeric fields are strings
      const numericFields = ['age', 'year', 'semester', 'year_of_admission', 'year_of_completion', 
        'curriculum_id', 'parent_income', 'amount', 'room_capacity', 'floor_no', 'nominee_age']
      
      numericFields.forEach(field => {
        if (formatted[field] !== undefined && formatted[field] !== null && formatted[field] !== '') {
          formatted[field] = String(formatted[field])
        }
      })
      
      // Ensure school_details items have numeric fields as strings and handle dates
      if (formatted.school_details && Array.isArray(formatted.school_details)) {
        formatted.school_details = formatted.school_details.map(school => ({
          ...school,
          year_of_pass: String(school.year_of_pass || ''),
          total_marks: String(school.total_marks || ''),
          tc_date: school.tc_date === '' ? null : school.tc_date
        }))
      }
      
      return formatted
    }

    const handleSubmit = async (e) => {
      e.preventDefault()
      setError('')
      setSuccess('')
      setLoading(true)

      try {
        const url = editingStudent 
          ? `${API_BASE_URL}/students/${editingStudent.student_id || editingStudent.id}`
          : `${API_BASE_URL}/students`
        const method = editingStudent ? 'PUT' : 'POST'
        const response = await fetch(url, {
          method: method,
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify(formatFormDataForSubmission(formData))
        })

        if (!response.ok) {
          throw new Error(editingStudent ? 'Failed to update student' : 'Failed to add student')
        }

        setSuccess(editingStudent ? 'Student updated successfully!' : 'Student added successfully!')
        
        // Reset form and state
        resetForm()
        setEditingStudent(null)
        
        // Refresh student list
        await fetchStudents()
        setShowForm(false)
      } catch (err) {
        setError(err.message)
      } finally {
        setLoading(false)
      }
    }

    // Convert DOB to YYYY-MM-DD format for HTML date input
    const formatDateForInput = (dateStr) => {
      if (!dateStr) return ''
      // If already in YYYY-MM-DD format, return as is
      if (/^\d{4}-\d{2}-\d{2}$/.test(dateStr)) return dateStr
      // Try to parse and reformat
      try {
        const date = new Date(dateStr)
        if (isNaN(date.getTime())) return ''
        const year = date.getFullYear()
        const month = String(date.getMonth() + 1).padStart(2, '0')
        const day = String(date.getDate()).padStart(2, '0')
        return `${year}-${month}-${day}`
      } catch (e) {
        return ''
      }
    }

    // Load student for editing
    const loadStudentForEdit = async (studentId) => {
      try {
        const res = await fetch(`${API_BASE_URL}/students/${studentId}`)
        if (!res.ok) throw new Error('Failed to load student')
        const data = await res.json()
        
        // Handle hierarchical response (FullStudent)
        const student = data.student || {}
        const academic = data.academic_details || {}
        const address = data.address || {}
        const payment = data.admission_payment || {}
        const contact = data.contact_details || {}
        const hostel = data.hostel_details || {}
        const insurance = data.insurance_details || {}
        const schools = data.school_details || []
        
        // Populate form with student data
        setFormData(prev => ({
          ...prev,
          // Basic
          enrollment_no: student.enrollment_no || '',
          register_no: student.register_no || '',
          dte_reg_no: student.dte_reg_no || '',
          application_no: student.application_no || '',
          admission_no: student.admission_no || '',
          student_name: student.student_name || '',
          gender: student.gender || '',
          dob: formatDateForInput(student.dob),
          age: String(student.age || ''),
          father_name: student.father_name || '',
          mother_name: student.mother_name || '',
          guardian_name: student.guardian_name || '',
          religion: student.religion || '',
          nationality: student.nationality || 'Indian',
          community: student.community || '',
          mother_tongue: student.mother_tongue || '',
          blood_group: student.blood_group || '',
          aadhar_no: student.aadhar_no || '',
          parent_occupation: student.parent_occupation || '',
          designation: student.designation || '',
          place_of_work: student.place_of_work || '',
          parent_income: String(student.parent_income || ''),

          // Academic
          batch: academic.batch || '',
          year: String(academic.year || ''),
          semester: String(academic.semester || ''),
          degree_level: academic.degree_level || '',
          section: academic.section || '',
          department: academic.department || '',
          student_category: academic.student_category || '',
          branch_type: academic.branch_type || '',
          seat_category: academic.seat_category || '',
          regulation: academic.regulation || '',
          quota: academic.quota || '',
          university: academic.university || '',
          year_of_admission: String(academic.year_of_admission || ''),
          year_of_completion: String(academic.year_of_completion || ''),
          student_status: academic.student_status || '',
          curriculum_id: String(academic.curriculum_id || ''),

          // Address
          permanent_address: address.permanent_address || '',
          present_address: address.present_address || '',
          residence_location: address.residence_location || '',

          // Payment
          dte_register_no: payment.dte_register_no || '',
          dte_admission_no: payment.dte_admission_no || '',
          receipt_no: payment.receipt_no || '',
          receipt_date: formatDateForInput(payment.receipt_date),
          amount: String(payment.amount || ''),
          bank_name: payment.bank_name || '',

          // Contact
          parent_mobile: contact.parent_mobile || '',
          student_mobile: contact.student_mobile || '',
          student_email: contact.student_email || '',
          parent_email: contact.parent_email || '',
          official_email: contact.official_email || '',

          // Hostel
          hosteller_type: hostel.hosteller_type || '',
          hostel_name: hostel.hostel_name || '',
          room_no: hostel.room_no || '',
          room_capacity: String(hostel.room_capacity || ''),
          room_type: hostel.room_type || '',
          floor_no: String(hostel.floor_no || ''),
          warden_name: hostel.warden_name || '',
          alternate_warden: hostel.alternate_warden || '',
          class_advisor: hostel.class_advisor || '',

          // Insurance
          nominee_name: insurance.nominee_name || '',
          relationship: insurance.relationship || '',
          nominee_age: String(insurance.nominee_age || ''),

          // Schools
          school_details: schools.length > 0 ? schools.map(s => ({
            school_name: s.school_name || '',
            board: s.board || '',
            year_of_pass: String(s.year_of_pass || ''),
            state: s.state || '',
            tc_no: s.tc_no || '',
            tc_date: formatDateForInput(s.tc_date),
            total_marks: String(s.total_marks || '')
          })) : [{ school_name: '', board: '', year_of_pass: '', state: '', tc_no: '', tc_date: '', total_marks: '' }]
        }))
        
        setEditingStudent(student)
        setShowForm(true)
        setError('')
      } catch (err) {
        setError('Failed to load student details: ' + err.message)
      }
    }

    const handleDelete = async (id) => {
      if (window.confirm('Are you sure you want to delete this student?')) {
        try {
          const res = await fetch(`${API_BASE_URL}/students/${id}`, {
            method: 'DELETE',
          })
          if (!res.ok) throw new Error('Failed to delete student')
          setSuccess('Student deleted successfully')
          fetchStudents()
        } catch (err) {
          console.error(err)
          setError(err.message)
        }
      }
    }

    // Reset form to initial state
    const resetForm = () => {
      setFormData({
        enrollment_no: '',
        register_no: '',
        dte_reg_no: '',
        application_no: '',
        admission_no: '',
        student_name: '',
        gender: '',
        dob: '',
        age: '',
        father_name: '',
        mother_name: '',
        guardian_name: '',
        religion: '',
        nationality: 'Indian',
        community: '',
        mother_tongue: '',
        blood_group: '',
        aadhar_no: '',
        parent_occupation: '',
        designation: '',
        place_of_work: '',
        parent_income: '',
        batch: '',
        year: '',
        semester: '',
        degree_level: '',
        section: '',
        department: '',
        student_category: '',
        branch_type: '',
        seat_category: '',
        regulation: '',
        quota: '',
        university: '',
        year_of_admission: '',
        year_of_completion: '',
        student_status: '',
        curriculum_id: '',
        permanent_address: '',
        present_address: '',
        residence_location: '',
        dte_register_no: '',
        dte_admission_no: '',
        receipt_no: '',
        receipt_date: '',
        amount: '',
        bank_name: '',
        parent_mobile: '',
        student_mobile: '',
        student_email: '',
        parent_email: '',
        official_email: '',
        hosteller_type: '',
        hostel_name: '',
        room_no: '',
        room_capacity: '',
        room_type: '',
        floor_no: '',
        warden_name: '',
        alternate_warden: '',
        class_advisor: '',
        nominee_name: '',
        relationship: '',
        nominee_age: '',
        school_details: [
          { school_name: '', board: '', year_of_pass: '', state: '', tc_no: '', tc_date: '', total_marks: '' }
        ]
      })
    }

    // Fetch list of students from API
    const fetchStudents = async () => {
      try {
        const res = await fetch(`${API_BASE_URL}/students`)
        if (!res.ok) throw new Error('Failed to fetch students')
        const data = await res.json()
        setStudents(Array.isArray(data) ? data : [])
      } catch (err) {
        console.error(err)
      }
    }

    // Fetch list of curriculums from API
    const fetchCurriculums = async () => {
      try {
        const res = await fetch(`${API_BASE_URL}/curriculum`)
        if (!res.ok) throw new Error('Failed to fetch curriculums')
        const data = await res.json()
        setCurriculums(Array.isArray(data) ? data : [])
      } catch (err) {
        console.error('Error fetching curriculums:', err)
      }
    }

    // Fetch list of departments from API
    const fetchDepartments = async () => {
      try {
        const res = await fetch(`${API_BASE_URL}/departments`)
        if (!res.ok) throw new Error('Failed to fetch departments')
        const data = await res.json()
        setDepartments(Array.isArray(data) ? data : [])
      } catch (err) {
        console.error('Error fetching departments:', err)
      }
    }

    useEffect(() => {
      fetchStudents()
      fetchCurriculums()
      fetchDepartments()
    }, [])

    return (
      <MainLayout
        title="Student Details"
        subtitle="Add and manage student information"
        actions={
          !showForm && (
            <div className="flex items-center space-x-3">
              <input
                type="search"
                placeholder="Search by name, student id, or enrollment..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="input-custom w-64"
              />
              <button
                type="button"
                onClick={() => setShowForm(true)}
                className="btn-primary-custom"
              >
                Create Student
              </button>
            </div>
          )
        }
      >
        <div className="in-w-screen mx-auto">
          {/* Messages */}
          {error && (
            <div className="mb-6 flex items-start space-x-3 p-4 bg-red-50 border border-red-200 rounded-lg">
              <svg className="w-5 h-5 text-red-600 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
              </svg>
              <p className="text-sm font-medium text-red-600">{error}</p>
            </div>
          )}

          {success && (
            <div className="mb-6 flex items-start space-x-3 p-4 bg-green-50 border border-green-200 rounded-lg">
              <svg className="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
              </svg>
              <p className="text-sm font-medium text-green-600">{success}</p>
            </div>
          )}

          {/* Student List (hidden when form shown) */}
          {!showForm && (
            <div className="mb-8">

            <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4">
              {students
                .filter(s => {
                  if (!searchTerm) return true
                  const q = searchTerm.toLowerCase()
                  const name = String(s.student_name || '').toLowerCase()
                  const id = String(s.student_id || s.id || '').toLowerCase()
                  const enroll = String(s.enrollment_no || '').toLowerCase()
                  return name.includes(q) || id.includes(q) || enroll.includes(q)
                })
                .map((s) => (
                  <StudentCard 
                    key={s.student_id || s.id} 
                    student={s} 
                    onEdit={loadStudentForEdit}
                    onDelete={handleDelete}
                  />
                ))}
            </div>
            </div>
          )}

          {/* Student Entry Form (toggleable) */}
          {showForm && (
            <div className="card-custom p-8">
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-2xl font-bold text-gray-900">
                {editingStudent ? 'Edit Student - ' + (editingStudent.student_name || 'Student') : 'Add New Student'}
              </h2>
              <button
                type="button"
                onClick={() => {
                  setShowForm(false)
                  setEditingStudent(null)
                  resetForm()
                }}
                className="text-gray-500 hover:text-gray-700 text-2xl"
              >
                ✕
              </button>
            </div>
            
            <form onSubmit={handleSubmit} className="space-y-8">
              {/* Identification Details */}
              <div>
                <h3 className="text-lg font-semibold text-gray-900 mb-4 pb-2 border-b border-gray-200">
                  Identification Details
                </h3>
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {/* Show Student ID only in edit mode (read-only) */}
                  {editingStudent && (
                    <div>
                      <label className="block text-sm font-semibold text-gray-700 mb-2">
                        Student ID (Auto-Generated)
                      </label>
                      <input
                        type="text"
                        value={editingStudent.student_id || '—'}
                        readOnly
                        disabled
                        className="input-custom bg-gray-100 cursor-not-allowed"
                      />
                    </div>
                  )}

                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Enrollment No <span className="text-red-500">*</span>
                    </label>
                    <input
                      type="text"
                      name="enrollment_no"
                      value={formData.enrollment_no}
                      onChange={handleInputChange}
                      required
                      className="input-custom"
                      placeholder="College enrollment number"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Register No <span className="text-red-500">*</span>
                    </label>
                    <input
                      type="text"
                      name="register_no"
                      value={formData.register_no}
                      onChange={handleInputChange}
                      required
                      className="input-custom"
                      placeholder="Registration number"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      DTE Reg No
                    </label>
                    <input
                      type="text"
                      name="dte_reg_no"
                      value={formData.dte_reg_no}
                      onChange={handleInputChange}
                      className="input-custom"
                      placeholder="DTE registration number"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Application No
                    </label>
                    <input
                      type="text"
                      name="application_no"
                      value={formData.application_no}
                      onChange={handleInputChange}
                      className="input-custom"
                      placeholder="Application number"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Admission No
                    </label>
                    <input
                      type="text"
                      name="admission_no"
                      value={formData.admission_no}
                      onChange={handleInputChange}
                      className="input-custom"
                      placeholder="Admission number"
                    />
                  </div>
                </div>
              </div>

              {/* Personal Details */}
              <div>
                <h3 className="text-lg font-semibold text-gray-900 mb-4 pb-2 border-b border-gray-200">
                  Personal Details
                </h3>
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  <div className="md:col-span-2">
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Student Name <span className="text-red-500">*</span>
                    </label>
                    <input
                      type="text"
                      name="student_name"
                      value={formData.student_name}
                      onChange={handleInputChange}
                      required
                      className="input-custom"
                      placeholder="Full name of student"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Gender
                    </label>
                    <select
                      name="gender"
                      value={formData.gender}
                      onChange={handleInputChange}
                      className="input-custom"
                    >
                      <option value="">Select Gender</option>
                      <option value="Male">Male</option>
                      <option value="Female">Female</option>
                      <option value="Other">Other</option>
                    </select>
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Date of Birth
                    </label>
                    <input
                      type="date"
                      name="dob"
                      value={formData.dob}
                      onChange={handleInputChange}
                      className="input-custom"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Age
                    </label>
                    <input
                      type="number"
                      name="age"
                      value={formData.age}
                      onChange={handleInputChange}
                      readOnly
                      className="input-custom bg-gray-50"
                      placeholder="Auto-calculated"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Blood Group
                    </label>
                    <select
                      name="blood_group"
                      value={formData.blood_group}
                      onChange={handleInputChange}
                      className="input-custom"
                    >
                      <option value="">Select Blood Group</option>
                      <option value="A+">A+</option>
                      <option value="A-">A-</option>
                      <option value="B+">B+</option>
                      <option value="B-">B-</option>
                      <option value="AB+">AB+</option>
                      <option value="AB-">AB-</option>
                      <option value="O+">O+</option>
                      <option value="O-">O-</option>
                    </select>
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Aadhar No
                    </label>
                    <input
                      type="text"
                      name="aadhar_no"
                      value={formData.aadhar_no}
                      onChange={handleInputChange}
                      maxLength="12"
                      pattern="[0-9]{12}"
                      className="input-custom"
                      placeholder="12-digit Aadhar number"
                    />
                  </div>
                        Aadhar No
                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Religion
                    </label>
                    <select
                      name="religion"
                      onChange={handleInputChange}
                      className="input-custom"
                    >
                      <option value="">Select Religion</option>
                      <option value="Hindu">Hindu</option>
                      <option value="Muslim">Muslim</option>
                      <option value="Christian">Christian</option>
                      <option value="Sikh">Sikh</option>
                      <option value="Buddhist">Buddhist</option>
                      <option value="Jain">Jain</option>
                      <option value="Other">Other</option>
                    </select>
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Nationality
                    </label>
                    <input
                      type="text"
                      name="nationality"
                      value={formData.nationality}
                      onChange={handleInputChange}
                      className="input-custom"
                      placeholder="e.g., Indian"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Community
                    </label>
                    <select
                      name="community"
                      value={formData.community}
                      onChange={handleInputChange}
                      className="input-custom"
                    >
                      <option value="">Select Community</option>
                      <option value="OC">OC (Open Category)</option>
                      <option value="BC">BC (Backward Class)</option>
                      <option value="MBC">MBC (Most Backward Class)</option>
                      <option value="SC">SC (Scheduled Caste)</option>
                      <option value="ST">ST (Scheduled Tribe)</option>
                      <option value="SCC">SCC (Scheduled Caste Convert)</option>
                    </select>
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Mother Tongue
                    </label>
                    <input
                      type="text"
                      name="mother_tongue"
                      value={formData.mother_tongue}
                      onChange={handleInputChange}
                      className="input-custom"
                      placeholder="Native language"
                    />
                  </div>
                </div>
              </div>

              {/* Family Details */}
              <div>
                <h3 className="text-lg font-semibold text-gray-900 mb-4 pb-2 border-b border-gray-200">
                  Family Details
                </h3>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Father's Name
                    </label>
                    <input
                      type="text"
                      name="father_name"
                      value={formData.father_name}
                      onChange={handleInputChange}
                      className="input-custom"
                      placeholder="Father's full name"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Mother's Name
                    </label>
                    <input
                      type="text"
                      name="mother_name"
                      value={formData.mother_name}
                      onChange={handleInputChange}
                      className="input-custom"
                      placeholder="Mother's full name"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Guardian's Name
                    </label>
                    <input
                      type="text"
                      name="guardian_name"
                      value={formData.guardian_name}
                      onChange={handleInputChange}
                      className="input-custom"
                      placeholder="Guardian's name (if applicable)"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Parent Occupation
                    </label>
                    <input
                      type="text"
                      name="parent_occupation"
                      value={formData.parent_occupation}
                      onChange={handleInputChange}
                      className="input-custom"
                      placeholder="Parent's occupation"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Designation
                    </label>
                    <input
                      type="text"
                      name="designation"
                      value={formData.designation}
                      onChange={handleInputChange}
                      className="input-custom"
                      placeholder="Parent's job designation"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Place of Work
                    </label>
                    <input
                      type="text"
                      name="place_of_work"
                      value={formData.place_of_work}
                      onChange={handleInputChange}
                      className="input-custom"
                      placeholder="Parent's workplace"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Annual Family Income
                    </label>
                    <input
                      type="number"
                      name="parent_income"
                      value={formData.parent_income}
                      onChange={handleInputChange}
                      className="input-custom"
                      placeholder="Annual income in ₹"
                    />
                  </div>
                </div>
              </div>

              {/* Academic Details Section */}
              <div>
                <h3 className="text-lg font-semibold text-gray-900 mb-4 pb-2 border-b border-gray-200">
                  Academic Details
                </h3>
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  <div>
                    <label className="block text-xs font-semibold text-gray-500 mb-1 ml-1">Batch <span className="text-red-500">*</span></label>
                    <input type="text" name="batch" placeholder="Batch" value={formData.batch} onChange={handleInputChange} required className="input-custom" />
                  </div>
                  <div>
                    <label className="block text-xs font-semibold text-gray-500 mb-1 ml-1">Year <span className="text-red-500">*</span></label>
                    <input type="number" name="year" placeholder="Year" value={formData.year} onChange={handleInputChange} required className="input-custom" />
                  </div>
                  <div>
                    <label className="block text-xs font-semibold text-gray-500 mb-1 ml-1">Semester <span className="text-red-500">*</span></label>
                    <input type="number" name="semester" placeholder="Semester" value={formData.semester} onChange={handleInputChange} required className="input-custom" />
                  </div>
                  <div>
                    <label className="block text-xs font-semibold text-gray-500 mb-1 ml-1">Degree Level</label>
                    <input type="text" name="degree_level" placeholder="Degree Level" value={formData.degree_level} onChange={handleInputChange} className="input-custom" />
                  </div>
                  <div>
                    <label className="block text-xs font-semibold text-gray-500 mb-1 ml-1">Section</label>
                    <input type="text" name="section" placeholder="Section" value={formData.section} onChange={handleInputChange} className="input-custom" />
                  </div>
                  <div>
                    <label className="block text-xs font-semibold text-gray-500 mb-1 ml-1">Department <span className="text-red-500">*</span></label>
                    <select
                      name="department"
                      value={formData.department}
                      onChange={handleInputChange}
                      required
                      className="input-custom"
                    >
                      <option value="">Select Department</option>
                      {departments.map(dept => (
                        <option key={dept.id} value={dept.department_name}>
                          {dept.department_name}
                        </option>
                      ))}
                    </select>
                  </div>
                  <input type="text" name="student_category" placeholder="Student Category" value={formData.student_category} onChange={handleInputChange} className="input-custom" />
                  <input type="text" name="branch_type" placeholder="Branch Type" value={formData.branch_type} onChange={handleInputChange} className="input-custom" />
                  <input type="text" name="seat_category" placeholder="Seat Category" value={formData.seat_category} onChange={handleInputChange} className="input-custom" />
                  <input type="text" name="regulation" placeholder="Regulation" value={formData.regulation} onChange={handleInputChange} className="input-custom" />
                  <input type="text" name="quota" placeholder="Quota" value={formData.quota} onChange={handleInputChange} className="input-custom" />
                  <input type="text" name="university" placeholder="University" value={formData.university} onChange={handleInputChange} className="input-custom" />
                  <input type="number" name="year_of_admission" placeholder="Year of Admission" value={formData.year_of_admission} onChange={handleInputChange} className="input-custom" />
                  <input type="number" name="year_of_completion" placeholder="Year of Completion" value={formData.year_of_completion} onChange={handleInputChange} className="input-custom" />
                  <input type="text" name="student_status" placeholder="Student Status" value={formData.student_status} onChange={handleInputChange} className="input-custom" />
                  <div className="flex flex-col">
                    <label className="block text-xs font-semibold text-gray-500 mb-1 ml-1">Curriculum <span className="text-red-500">*</span></label>
                    <SearchBarWithDropdown
                      value={curriculumDisplay}
                      onChange={(e) => setCurriculumDisplay(e.target.value)}
                      items={curriculums}
                      onSelect={(item) => {
                        setFormData(prev => ({...prev, curriculum_id: item.id}))
                        setCurriculumDisplay(`${item.name} (${item.academic_year})`)
                      }}
                      placeholder="Select Curriculum"
                      renderItem={(item) => <div>{item.name} ({item.academic_year})</div>}
                      getItemKey={(item) => item.id}
                    />
                  </div>
                </div>
              </div>

              {/* Address Details Section */}
              <div>
                <h3 className="text-lg font-semibold text-gray-900 mb-4 pb-2 border-b border-gray-200">
                  Address Details (Optional)
                </h3>
                <div className="grid grid-cols-1 gap-4">
                  <div>
                    <label className="block text-xs font-semibold text-gray-500 mb-1 ml-1">Permanent Address</label>
                    <textarea name="permanent_address" placeholder="Permanent Address" value={formData.permanent_address} onChange={handleInputChange} rows="2" className="input-custom" />
                  </div>
                  <textarea name="present_address" placeholder="Present Address" value={formData.present_address} onChange={handleInputChange} rows="2" className="input-custom" />
                  <input type="text" name="residence_location" placeholder="Residence Location" value={formData.residence_location} onChange={handleInputChange} className="input-custom" />
                </div>
              </div>

              {/* Contact Details Section */}
              <div>
                <h3 className="text-lg font-semibold text-gray-900 mb-4 pb-2 border-b border-gray-200">
                  Contact Details (Optional)
                </h3>
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  <div>
                    <label className="block text-xs font-semibold text-gray-500 mb-1 ml-1">Parent Mobile</label>
                    <input type="tel" name="parent_mobile" placeholder="Parent Mobile" value={formData.parent_mobile} onChange={handleInputChange} className="input-custom" />
                  </div>
                  <input type="tel" name="student_mobile" placeholder="Student Mobile" value={formData.student_mobile} onChange={handleInputChange} className="input-custom" />
                  <div>
                    <label className="block text-xs font-semibold text-gray-500 mb-1 ml-1">Student Email</label>
                    <input type="email" name="student_email" placeholder="Student Email" value={formData.student_email} onChange={handleInputChange} className="input-custom" />
                  </div>
                  <input type="email" name="parent_email" placeholder="Parent Email" value={formData.parent_email} onChange={handleInputChange} className="input-custom" />
                  <input type="email" name="official_email" placeholder="Official Email" value={formData.official_email} onChange={handleInputChange} className="input-custom" />
                </div>
              </div>

              {/* Admission Payment Section */}
              <div>
                <h3 className="text-lg font-semibold text-gray-900 mb-4 pb-2 border-b border-gray-200">
                  Admission Payment (Optional)
                </h3>
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  <input type="text" name="dte_register_no" placeholder="DTE Register No" value={formData.dte_register_no} onChange={handleInputChange} className="input-custom" />
                  <input type="text" name="dte_admission_no" placeholder="DTE Admission No" value={formData.dte_admission_no} onChange={handleInputChange} className="input-custom" />
                  <input type="text" name="receipt_no" placeholder="Receipt No" value={formData.receipt_no} onChange={handleInputChange} className="input-custom" />
                  <input type="date" name="receipt_date" placeholder="Receipt Date" value={formData.receipt_date} onChange={handleInputChange} className="input-custom" />
                  <input type="number" name="amount" placeholder="Amount" value={formData.amount} onChange={handleInputChange} className="input-custom" step="0.01" />
                  <input type="text" name="bank_name" placeholder="Bank Name" value={formData.bank_name} onChange={handleInputChange} className="input-custom" />
                </div>
              </div>

              {/* Hostel Details Section */}
              <div>
                <h3 className="text-lg font-semibold text-gray-900 mb-4 pb-2 border-b border-gray-200">
                  Hostel Details (Optional)
                </h3>
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  <input type="text" name="hosteller_type" placeholder="Hosteller Type" value={formData.hosteller_type} onChange={handleInputChange} className="input-custom" />
                  <input type="text" name="hostel_name" placeholder="Hostel Name" value={formData.hostel_name} onChange={handleInputChange} className="input-custom" />
                  <input type="text" name="room_no" placeholder="Room No" value={formData.room_no} onChange={handleInputChange} className="input-custom" />
                  <input type="number" name="room_capacity" placeholder="Room Capacity" value={formData.room_capacity} onChange={handleInputChange} className="input-custom" />
                  <input type="text" name="room_type" placeholder="Room Type" value={formData.room_type} onChange={handleInputChange} className="input-custom" />
                  <input type="number" name="floor_no" placeholder="Floor No" value={formData.floor_no} onChange={handleInputChange} className="input-custom" />
                  <input type="text" name="warden_name" placeholder="Warden Name" value={formData.warden_name} onChange={handleInputChange} className="input-custom" />
                  <input type="text" name="alternate_warden" placeholder="Alternate Warden" value={formData.alternate_warden} onChange={handleInputChange} className="input-custom" />
                  <input type="text" name="class_advisor" placeholder="Class Advisor" value={formData.class_advisor} onChange={handleInputChange} className="input-custom" />
                </div>
              </div>

              {/* Insurance Details Section */}
              <div>
                <h3 className="text-lg font-semibold text-gray-900 mb-4 pb-2 border-b border-gray-200">
                  Insurance Details (Optional)
                </h3>
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  <input type="text" name="nominee_name" placeholder="Nominee Name" value={formData.nominee_name} onChange={handleInputChange} className="input-custom" />
                  <input type="text" name="relationship" placeholder="Relationship" value={formData.relationship} onChange={handleInputChange} className="input-custom" />
                  <input type="number" name="nominee_age" placeholder="Nominee Age" value={formData.nominee_age} onChange={handleInputChange} className="input-custom" />
                </div>
              </div>

              {/* School Details Section */}
              <div>
                <h3 className="text-lg font-semibold text-gray-900 mb-4 pb-2 border-b border-gray-200">
                  School Details (Optional - Multiple schools)
                </h3>
                {formData.school_details && formData.school_details.map((school, idx) => (
                  <div key={idx} className="mb-4 p-4 border border-gray-200 rounded-lg">
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                      <input 
                        type="text" 
                        placeholder="School Name" 
                        value={school.school_name} 
                        onChange={(e) => {
                          const newSchools = [...formData.school_details];
                          newSchools[idx].school_name = e.target.value;
                          setFormData({...formData, school_details: newSchools});
                        }}
                        className="input-custom" 
                      />
                      <input 
                        type="text" 
                        placeholder="Board" 
                        value={school.board} 
                        onChange={(e) => {
                          const newSchools = [...formData.school_details];
                          newSchools[idx].board = e.target.value;
                          setFormData({...formData, school_details: newSchools});
                        }}
                        className="input-custom" 
                      />
                      <input 
                        type="number" 
                        placeholder="Year of Pass" 
                        value={school.year_of_pass} 
                        onChange={(e) => {
                          const newSchools = [...formData.school_details];
                          newSchools[idx].year_of_pass = e.target.value;
                          setFormData({...formData, school_details: newSchools});
                        }}
                        className="input-custom" 
                      />
                      <input 
                        type="text" 
                        placeholder="State" 
                        value={school.state} 
                        onChange={(e) => {
                          const newSchools = [...formData.school_details];
                          newSchools[idx].state = e.target.value;
                          setFormData({...formData, school_details: newSchools});
                        }}
                        className="input-custom" 
                      />
                      <input 
                        type="text" 
                        placeholder="TC No" 
                        value={school.tc_no} 
                        onChange={(e) => {
                          const newSchools = [...formData.school_details];
                          newSchools[idx].tc_no = e.target.value;
                          setFormData({...formData, school_details: newSchools});
                        }}
                        className="input-custom" 
                      />
                      <div className="flex flex-col">
                        <label className="text-xs text-gray-500 mb-1 ml-1">TC Date</label>
                        <input 
                          type="date" 
                          placeholder="TC Date" 
                          value={school.tc_date} 
                          onChange={(e) => {
                            const newSchools = [...formData.school_details];
                            newSchools[idx].tc_date = e.target.value;
                            setFormData({...formData, school_details: newSchools});
                          }}
                          className="input-custom" 
                        />
                      </div>
                      <div className="flex flex-col">
                        <label className="text-xs text-gray-500 mb-1 ml-1">Total Marks</label>
                        <input 
                          type="number" 
                          placeholder="Total Marks" 
                          value={school.total_marks} 
                          onChange={(e) => {
                            const newSchools = [...formData.school_details];
                            newSchools[idx].total_marks = e.target.value;
                            setFormData({...formData, school_details: newSchools});
                          }}
                          className="input-custom" 
                          step="0.01"
                        />
                      </div>
                      
                    </div>
                    {formData.school_details.length > 1 && (
                      <button
                        type="button"
                        onClick={() => {
                          const newSchools = formData.school_details.filter((_, i) => i !== idx);
                          setFormData({...formData, school_details: newSchools});
                        }}
                        className="mt-2 px-4 py-2 text-sm text-red-600 border border-red-300 rounded hover:bg-red-50"
                      >
                        Remove School
                      </button>
                    )}
                  </div>
                ))}
                <button
                  type="button"
                  onClick={() => {
                    setFormData({
                      ...formData,
                      school_details: [...formData.school_details, { school_name: '', board: '', year_of_pass: '', state: '', tc_no: '', tc_date: '', total_marks: '' }]
                    });
                  }}
                  className="px-4 py-2 text-sm text-blue-600 border border-blue-300 rounded hover:bg-blue-50"
                >
                  + Add Another School
                </button>
              </div>
              <div className="flex justify-end space-x-4 pt-4">
                <button
                  type="button"
                  onClick={() => setFormData({
                    enrollment_no: '',
                    register_no: '',
                    dte_reg_no: '',
                    application_no: '',
                    admission_no: '',
                    student_name: '',
                    gender: '',
                    dob: '',
                    age: '',
                    father_name: '',
                    mother_name: '',
                    guardian_name: '',
                    religion: '',
                    nationality: 'Indian',
                    community: '',
                    mother_tongue: '',
                    blood_group: '',
                    aadhar_no: '',
                    parent_occupation: '',
                    designation: '',
                    place_of_work: '',
                    parent_income: '',
                    batch: '',
                    year: '',
                    semester: '',
                    degree_level: '',
                    section: '',
                    department: '',
                    student_category: '',
                    branch_type: '',
                    seat_category: '',
                    regulation: '',
                    quota: '',
                    university: '',
                    year_of_admission: '',
                    year_of_completion: '',
                    student_status: '',
                    curriculum_id: '',
                    permanent_address: '',
                    present_address: '',
                    residence_location: '',
                    dte_register_no: '',
                    dte_admission_no: '',
                    receipt_no: '',
                    receipt_date: '',
                    amount: '',
                    bank_name: '',
                    parent_mobile: '',
                    student_mobile: '',
                    student_email: '',
                    parent_email: '',
                    official_email: '',
                    hosteller_type: '',
                    hostel_name: '',
                    room_no: '',
                    room_capacity: '',
                    room_type: '',
                    floor_no: '',
                    warden_name: '',
                    alternate_warden: '',
                    class_advisor: '',
                    nominee_name: '',
                    relationship: '',
                    nominee_age: '',
                    school_details: [
                      { school_name: '', board: '', year_of_pass: '', state: '', tc_no: '', tc_date: '', total_marks: '' }
                    ]
                  })}
                  className="px-6 py-2.5 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 font-medium transition-colors"
                >
                  Reset
                </button>
                <button
                  type="button"
                  onClick={() => {
                    setShowForm(false)
                    setEditingStudent(null)
                    resetForm()
                  }}
                  className="px-6 py-2.5 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 font-medium transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={loading}
                  className="btn-primary-custom disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {editingStudent 
                    ? (loading ? 'Updating Student...' : 'Update Student')
                    : (loading ? 'Adding Student...' : 'Add Student')
                  }
                </button>
              </div>
            </form>
          </div>
          )}
        </div>
      </MainLayout>
    )
  }

  export default StudentDetailsPage
