import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import MainLayout from '../../components/MainLayout'
import { API_BASE_URL } from '../../config'
import StatCard from '../../components/StatCard'
import QuickActionBtn from '../../components/QuickActionBtn'

function Dashboard() {
  const navigate = useNavigate()
  const userRole = localStorage.getItem('userRole')

  const [stats, setStats] = useState({
    totalCurriculum: 0,
    activeCurriculum: 0,
    totalCourses: 0,
    recentActivities: 0
  })

  const [markEntryStats, setMarkEntryStats] = useState({
    totalWindows: 0,
    activeWindows: 0,
    upcomingWindows: 0,
    teachersWithPermissions: 0
  })

  useEffect(() => {
    if (userRole === 'teacher') {
      navigate('/teacher/course-selection');
    } else if (userRole === 'student') {
      navigate('/student/elective-selection');
    } else if (userRole === 'hr') {
      navigate('/hr/faculty');
    }
  }, [navigate]);

  useEffect(() => {
    fetchDashboardStats()
    if (userRole === 'coe') {
      fetchMarkEntryStats()
    }
  }, [])

  const fetchDashboardStats = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/curriculum`)
      if (response.ok) {
        const data = await response.json()
        setStats({
          totalCurriculum: data.length || 0,
          activeCurriculum: data.length || 0,
          totalCourses: 0,
          recentActivities: 0
        })
      } else {
        setStats({ totalCurriculum: 0, activeCurriculum: 0, totalCourses: 0, recentActivities: 0 })
      }
    } catch (error) {
      console.error('Error fetching dashboard stats:', error)
      setStats({ totalCurriculum: 0, activeCurriculum: 0, totalCourses: 0, recentActivities: 0 })
    }
  }

  const fetchMarkEntryStats = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/mark-entry/stats`)
      if (response.ok) {
        const data = await response.json()
        setMarkEntryStats({
          totalWindows: data.total_windows || 0,
          activeWindows: data.active_windows || 0,
          upcomingWindows: data.upcoming_windows || 0,
          teachersWithPermissions: data.teachers_with_permissions || 0
        })
      }
    } catch (error) {
      console.error('Error fetching mark entry stats:', error)
    }
  }

  const statCards = userRole === 'coe' ? [
    {
      title: 'Total Windows',
      value: markEntryStats.totalWindows,
      icon: (
        <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
        </svg>
      ),
    },
    {
      title: 'Active Windows',
      value: markEntryStats.activeWindows,
      icon: (
        <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      ),
    },
    {
      title: 'Upcoming Windows',
      value: markEntryStats.upcomingWindows,
      icon: (
        <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      ),
    },
    {
      title: 'Teachers with Permissions',
      value: markEntryStats.teachersWithPermissions,
      icon: (
        <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
        </svg>
      ),
    }
  ] : [
    {
      title: 'Total Curriculum',
      value: stats.totalCurriculum,
      icon: (
        <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
        </svg>
      ),
    },
    {
      title: 'Active Curriculum',
      value: stats.activeCurriculum,
      icon: (
        <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
        </svg>
      ),
    },
    {
      title: 'Total Courses',
      value: stats.totalCourses,
      icon: (
        <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
        </svg>
      ),
    },
    {
      title: 'Recent Activities',
      value: stats.recentActivities,
      icon: (
        <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      ),
    }
  ]

  const quickActions = [
    {
      title: 'View Curriculum',
      description: 'Browse all curriculum structures',
      icon: (
        <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
        </svg>
      ),
      action: () => navigate('/curriculum')
    },
    {
      title: 'Manage Clusters',
      description: 'Create and manage department clusters',
      icon: (
        <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
        </svg>
      ),
      action: () => navigate('/clusters')
    },
    {
      title: 'Manage Sharing',
      description: 'Control content sharing between cluster departments',
      icon: (
        <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
        </svg>
      ),
      action: () => navigate('/sharing')
    }
  ]

  // Add Users Management action for admin users only
  if (userRole === 'admin') {
    quickActions.push({
      title: 'User Management',
      description: 'Manage system users and permissions',
      icon: (
        <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
        </svg>
      ),
      action: () => navigate('/users')
    })
  }

  return (
    <MainLayout
      title="Dashboard"
      subtitle="Welcome back! Here's what's happening with your curriculum"
    >
      <div className="space-y-8">
        {/* Stats Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-x-4">
          {statCards.map((stat, index) => (
            <StatCard key={index} stat={stat} />
          ))}
        </div>

        {/* Quick Actions */}
        <div className="bg-white rounded-lg shadow-sm border border-primary-200 overflow-hidden ">
          <div className="px-6 py-4 border-b border-gray-200">
            <h2 className="text-xl font-bold text-gray-900">Quick Actions</h2>
            <p className="text-sm text-gray-600 mt-1">
              Access frequently used features
            </p>
          </div>
          <div className="p-6 grid grid-cols-1 lg:grid-cols-2 gap-6">
            {quickActions.map((action, index) => (
              <QuickActionBtn key={index} action={action} />
            ))}
          </div>
        </div>

        {/* Welcome Message */}
        <div className="card-custom p-8 text-white" style={{background: 'linear-gradient(to bottom right, rgb(67, 113, 229), rgb(47, 93, 209))'}}>
          <div className="flex items-start justify-between">
            <div className="flex-1">
              <h2 className="text-2xl font-bold mb-3">Welcome to Curriculum Management System</h2>
              <p className="mb-6 max-w-2xl" style={{color: 'rgba(255, 255, 255, 0.9)'}}>
                Streamline your academic planning with our comprehensive curriculum management platform. 
                Create, manage, and track curriculum structures, courses, and mappings all in one place.
              </p>
              <button
                onClick={() => navigate('/curriculum')}
                className="bg-white px-6 py-3 rounded-lg font-semibold hover:shadow-lg transition-all duration-200 hover:scale-105 active:scale-95"
                style={{color: 'rgb(67, 113, 229)'}}
              >
                Get Started
              </button>
            </div>
            <div className="hidden lg:block">
              <svg className="w-32 h-32 opacity-50" style={{color: 'rgba(255, 255, 255, 0.4)'}} fill="currentColor" viewBox="0 0 20 20">
                <path d="M10.394 2.08a1 1 0 00-.788 0l-7 3a1 1 0 000 1.84L5.25 8.051a.999.999 0 01.356-.257l4-1.714a1 1 0 11.788 1.838L7.667 9.088l1.94.831a1 1 0 00.787 0l7-3a1 1 0 000-1.838l-7-3zM3.31 9.397L5 10.12v4.102a8.969 8.969 0 00-1.05-.174 1 1 0 01-.89-.89 11.115 11.115 0 01.25-3.762zM9.3 16.573A9.026 9.026 0 007 14.935v-3.957l1.818.78a3 3 0 002.364 0l5.508-2.361a11.026 11.026 0 01.25 3.762 1 1 0 01-.89.89 8.968 8.968 0 00-5.35 2.524 1 1 0 01-1.4 0zM6 18a1 1 0 001-1v-2.065a8.935 8.935 0 00-2-.712V17a1 1 0 001 1z" />
              </svg>
            </div>
          </div>
        </div>
      </div>
    </MainLayout>
  )
}

export default Dashboard