import React from 'react'
import { useNavigate } from 'react-router-dom'
import MainLayout from '../../components/MainLayout'
import { useState } from 'react'

function TeacherStudentDashboard() {
  const navigate = useNavigate()
  const [hoveredIndex, setHoveredIndex] = useState(null)

  const actions = [
    {
      title: 'Teacher Details',
      description: 'Manage teacher profiles and information',
      icon: (
        <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10a2 2 0 012 2v1m2 13a2 2 0 01-2-2V7m2 13a2 2 0 002-2V9a2 2 0 00-2-2h-2m-4-3H9M7 16h6M7 8h6v4H7V8z" />
        </svg>
      ),
      action: () => navigate('/teacher-details')
    },
    {
      title: 'Student Details',
      description: 'View and manage student records',
      icon: (
        <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
        </svg>
      ),
      action: () => navigate('/Student_details')
    }
  ]

  return (
    <MainLayout
      title="Student Teacher Entry"
      subtitle="Manage teachers and students"
    >
      <div className="space-y-8">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {actions.map((action, index) => (
            <button
              key={index}
              onClick={action.action}
              onMouseEnter={() => setHoveredIndex(index)}
              onMouseLeave={() => setHoveredIndex(null)}
              className="flex flex-col items-center text-center p-8 bg-white rounded-xl card-custom transition-all duration-200 border hover:border-primary hover:border-primary-400 group relative"
            >
              {hoveredIndex === index && (
                <div className="absolute -top-2 left-1/2 -translate-x-1/2 bg-primary text-white px-3 py-1.5 rounded-md text-xs font-medium shadow-lg opacity-0 translate-y-1 group-hover:opacity-100 group-hover:translate-y-0 transition-all duration-200 pointer-events-none z-20">
                  Click To View
                </div>
              )}
              <div className="w-16 h-16 text-primary bg-primary-100 border border-primary rounded-full flex items-center justify-center mb-4 transition-colors duration-200">
                {action.icon}
              </div>
              <h3 className="text-xl font-bold text-gray-900 mb-2">{action.title}</h3>
              <p className="text-gray-500">{action.description}</p>
            </button>
          ))}
        </div>
      </div>
    </MainLayout>
  )
}

export default TeacherStudentDashboard
