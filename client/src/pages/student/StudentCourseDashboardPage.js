import React, { useEffect, useMemo, useState } from 'react';
import MainLayout from '../../components/MainLayout';
import { API_BASE_URL } from '../../config';

const StudentCourseDashboardPage = () => {
  const userEmail = localStorage.getItem('userEmail');

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [dashboard, setDashboard] = useState(null);

  useEffect(() => {
    const fetchDashboard = async () => {
      if (!userEmail) {
        setError('Unable to identify student account. Please login again.');
        setLoading(false);
        return;
      }

      try {
        setLoading(true);
        const res = await fetch(`${API_BASE_URL}/students/course-dashboard?email=${encodeURIComponent(userEmail)}`);
        if (!res.ok) {
          const txt = await res.text();
          throw new Error(txt || 'Failed to fetch student course dashboard');
        }
        const data = await res.json();
        setDashboard(data);
      } catch (err) {
        setError(err.message || 'Failed to load dashboard');
      } finally {
        setLoading(false);
      }
    };

    fetchDashboard();
  }, [userEmail]);

  const typeBadgeClass = (type) => {
    const key = String(type || '').toUpperCase();
    if (key === 'CORE') return 'bg-gray-100 text-gray-700';
    if (key === 'OPEN') return 'bg-green-100 text-green-700';
    if (key === 'PROFESSIONAL') return 'bg-blue-100 text-blue-700';
    if (key === 'HONOR') return 'bg-yellow-100 text-yellow-700';
    if (key === 'MINOR') return 'bg-orange-100 text-orange-700';
    if (key === 'ADDON') return 'bg-purple-100 text-purple-700';
    return 'bg-gray-100 text-gray-700';
  };

  const totals = useMemo(() => {
    if (!dashboard?.semesters) return { courses: 0, credits: 0 };
    return dashboard.semesters.reduce(
      (acc, sem) => {
        acc.courses += Number(sem.course_count || 0);
        acc.credits += Number(sem.total_credit || 0);
        return acc;
      },
      { courses: 0, credits: 0 }
    );
  }, [dashboard]);

  if (loading) {
    return (
      <MainLayout title="My Course Dashboard" subtitle="Semester-wise course history and selections">
        <div className="flex items-center justify-center min-h-[40vh]">
          <div className="flex flex-col items-center gap-3">
            <div className="w-10 h-10 border-4 border-gray-300 border-t-primary rounded-full animate-spin" />
            <p className="text-gray-600 text-sm">Loading dashboard…</p>
          </div>
        </div>
      </MainLayout>
    );
  }

  return (
    <MainLayout
      title="My Course Dashboard"
      subtitle={`Semester 1 to ${dashboard?.next_semester || '-'} (Current: ${dashboard?.current_semester || '-'})`}
    >
      <div className="space-y-4">
        {error && (
          <div className="bg-red-50 border border-red-200 text-red-700 rounded-lg p-3 text-sm">{error}</div>
        )}

        {!error && dashboard && (
          <>
            <div className="bg-white border rounded-lg p-4 flex flex-wrap items-center gap-3 text-sm">
              <span className="font-semibold text-gray-900">Total Courses: {totals.courses}</span>
              <span className="text-gray-400">•</span>
              <span className="font-semibold text-gray-900">Total Credits: {totals.credits}</span>
              <span className="text-gray-400">•</span>
              <span className="text-gray-600">Includes Core + Professional + Open + Honor + Minor + Addon</span>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
              {(dashboard.semesters || []).map((sem) => (
                <div
                  key={sem.semester}
                  className={`rounded-lg border p-4 ${
                    sem.semester === dashboard.next_semester
                      ? 'border-primary/40 bg-blue-50'
                      : 'border-gray-200 bg-white'
                  }`}
                >
                  <div className="flex items-center justify-between mb-2">
                    <h3 className="text-sm font-bold text-gray-900">Semester {sem.semester}</h3>
                    <span className="text-xs px-2 py-0.5 rounded-full bg-gray-100 text-gray-700 font-semibold">
                      {sem.course_count} course{sem.course_count === 1 ? '' : 's'}
                    </span>
                  </div>

                  <p className="text-xs text-gray-500 mb-3">Credits: <span className="font-semibold text-gray-800">{sem.total_credit}</span></p>

                  {Array.isArray(sem.courses) && sem.courses.length > 0 ? (
                    <div className="space-y-2 max-h-56 overflow-y-auto pr-1">
                      {sem.courses.map((course, idx) => (
                        <div key={`${sem.semester}-${course.course_id}-${idx}`} className="rounded border border-gray-100 bg-white px-2 py-1.5">
                          <div className="flex items-center gap-2 mb-0.5">
                            <span className={`text-[10px] font-semibold px-1.5 py-0.5 rounded ${typeBadgeClass(course.slot_type)}`}>
                              {course.slot_type}
                            </span>
                            <span className="text-[10px] text-gray-400 uppercase">{course.source}</span>
                          </div>
                          <p className="text-xs font-semibold text-gray-900">{course.course_code} — {course.course_name}</p>
                          <p className="text-[11px] text-gray-500">{course.category || 'General'} · {course.credits} credits</p>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p className="text-xs text-gray-400 italic">No course data in this semester</p>
                  )}
                </div>
              ))}
            </div>
          </>
        )}
      </div>
    </MainLayout>
  );
};

export default StudentCourseDashboardPage;
