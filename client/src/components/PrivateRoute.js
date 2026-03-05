import React from 'react'
import { Navigate, Outlet } from 'react-router-dom'

function PrivateRoute() {
  const userRole = localStorage.getItem('userRole')
  const userId = localStorage.getItem('userId')

  if (!userRole || !userId) {
    return <Navigate to="/" replace />
  }

  return <Outlet />
}

export default PrivateRoute


