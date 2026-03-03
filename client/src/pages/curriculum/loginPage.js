import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { API_BASE_URL } from '../../config'

function LoginPage() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const navigate = useNavigate()

  const handleLogin = async (e) => {
    e.preventDefault()
    setIsLoading(true)
    setError('')
    try {
      const response = await fetch(`${API_BASE_URL}/auth/login`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ username, password }),
      })

      const data = await response.json();

      if (data.success) {
        // Determine the best name to display
        const displayName = data.teacher_data?.name || data.user.username;
        
        // Store user info in localStorage
        localStorage.setItem('userRole', data.user.role)
        localStorage.setItem('userName', data.user.full_name || data.teacher_name || data.user.username)
        localStorage.setItem('userEmail', data.user.email)
        localStorage.setItem('userId', data.user.id)
        localStorage.setItem('user_id', data.user.id) // Store with underscore for consistency
        localStorage.setItem('username', data.user.username) // Store username for API calls
        
        // Store teacher ID if teacher role (use teachers.id, NOT users.id)
        if (data.user.role === 'teacher' && data.teacher_data?.teacher_id) {
          localStorage.setItem('teacherId', data.teacher_data.teacher_id)
          localStorage.setItem('teacher_id', data.teacher_data.teacher_id) // This is teachers.id, not users.id
          localStorage.setItem('faculty_id', data.teacher_data.faculty_id) // Store alphanumeric faculty ID too
        }

        setUsername("");
        setPassword("");

        // Redirect based on role
        if (data.user.role === "teacher") {
          navigate("/teacher-dashboard");
        } else if (data.user.role === "curriculum_entry_user") {
          navigate("/curriculum");
        } else if (data.user.role === "hod") {
          navigate("/hod/elective-management");
        } else {
          // Clear any stale teacher data from previous sessions
          localStorage.removeItem('teacher_id');
          localStorage.removeItem('faculty_id');
          localStorage.removeItem('teacher_name');
          localStorage.removeItem('teacher_email');
          localStorage.removeItem('teacher_dept');
          localStorage.removeItem('teacher_designation');
          console.log('No teacher data - cleared stale teacher localStorage');
        }
        
        console.log('Login successful, stored email:', data.user.email);
        
        setUsername('')
        setPassword('')
        
        // Navigate to first available page for the user's role
        const role = data.user.role;
        const roleRoutes = {
          'admin': '/dashboard',
          'hod': '/curriculum',
          'hr': '/hr/faculty',
          'teacher': '/teacher/course-selection',
          'student': '/student/elective-selection'
        };
        
        navigate(roleRoutes[role] || '/dashboard')
      } else {
        setError(data.message || 'Invalid username or password')
        setIsLoading(false)
      }
    } catch (err) {
      console.error('Login error:', err)
      setError('Failed to connect to server. Please try again.')
      setIsLoading(false)
    }
  }

  return (
    <div
      className="min-h-screen bg-background flex items-center justify-center p-4"
    >
      {/* Background decorative elements */}
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div className="absolute -top-40 -right-40 w-80 h-80 rounded-full mix-blend-multiply filter blur-xl opacity-70 animate-blob" style={{backgroundColor: 'rgba(67, 113, 229, 0.3)'}}></div>
        <div className="absolute -bottom-40 -left-40 w-80 h-80 bg-purple-200 rounded-full mix-blend-multiply filter blur-xl opacity-70 animate-blob animation-delay-2000"></div>
        <div className="absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 w-80 h-80 rounded-full mix-blend-multiply filter blur-xl opacity-70 animate-blob animation-delay-4000" style={{backgroundColor: 'rgba(67, 113, 229, 0.4)'}}></div>
      </div>

      <div className="relative w-full max-w-md">
        {/* Logo and Header */}
        <div className="text-center mb-8">
          <div
            className="inline-flex items-center justify-center w-20 h-20 rounded-3xl shadow-2xl mb-6 transform hover:scale-105 transition-transform"
            style={{
              backgroundColor: "#7D53F6",
            }}
          >
            <svg
              className="w-11 h-11 text-white"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253"
              />
            </svg>
          </div>
          <h1
            className="text-4xl font-bold mb-2"
            style={{
              color: "#7D53F6",
            }}
          >
            Curriculum Management System
          </h1>
          <p className="text-gray-600 text-base font-medium">
            Sign in to continue to your dashboard
          </p>
        </div>

        {/* Login Card */}
        <div className="bg-white rounded-2xl shadow-xl border border-gray-100 p-8">
          <form onSubmit={handleLogin} className="space-y-6">
            {/* Username */}
            <div>
              <label htmlFor="username" className="block text-sm font-semibold text-gray-700 mb-2">
                Username
              </label>
              <div className="relative">
                <div className="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
                  <svg className="h-5 w-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                  </svg>
                </div>
                <input
                  type="text"
                  id="username"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  placeholder="Enter your username"
                  required
                  disabled={isLoading}
                  className="input-custom pl-12 disabled:bg-gray-50 disabled:text-gray-500"
                />
              </div>
            </div>

            {/* Password */}
            <div>
              <label htmlFor="password" className="block text-sm font-semibold text-gray-700 mb-2">
                Password
              </label>
              <div className="relative">
                <div className="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
                  <svg className="h-5 w-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                  </svg>
                </div>
                <input
                  type="password"
                  id="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="Enter your password"
                  required
                  disabled={isLoading}
                  className="input-custom pl-12 disabled:bg-gray-50 disabled:text-gray-500"
                />
              </div>
            </div>

            {/* Error Message */}
            {error && (
              <div className="flex items-start space-x-3 p-4 bg-red-50 border border-red-200 rounded-lg">
                <svg className="w-5 h-5 text-red-600 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
                </svg>
                <div>
                  <p className="text-sm font-medium text-red-600">{error}</p>
                </div>
              </div>
            )}

            {/* Submit Button */}
            <button
              type="submit"
              disabled={isLoading}
              className="w-full btn-primary-custom disabled:opacity-70 disabled:cursor-not-allowed flex items-center justify-center space-x-2"
            >
              {isLoading ? (
                <>
                  <svg className="animate-spin h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                  </svg>
                  <span>Signing in...</span>
                </>
              ) : (
                <>
                  <span>Sign In</span>
                  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14 5l7 7m0 0l-7 7m7-7H3" />
                  </svg>
                </>
              )}
            </button>
          </form>
        </div>

        {/* Footer */}
        <p className="text-center text-sm text-gray-500 mt-6">
          © 2025 Curriculum Management System. All rights reserved.
        </p>
      </div>
    </div>
  )
}

export default LoginPage;
