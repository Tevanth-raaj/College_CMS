import React, { useEffect, useMemo, useRef, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { API_BASE_URL } from "../config";

const COLLAPSED_WIDTH_PX = 80; // Tailwind w-20
const EXPANDED_WIDTH_PX = 256; // Tailwind w-64

function Sidebar({ onExpandedChange }) {
  const navigate = useNavigate();
  const location = useLocation();

  // "Pinned" means keep expanded without hover.
  const [sidebarPinned, setSidebarPinned] = useState(false);

  // Hover state is handled via hysteresis based on mouse X.
  const [isHovering, setIsHovering] = useState(false);

  // Prevent collapse while clicking/dragging in sidebar.
  const [isPointerDownInSidebar, setIsPointerDownInSidebar] = useState(false);

  // Track if user has assigned windows
  const [hasAssignedWindows, setHasAssignedWindows] = useState(false);
  const [hasCheckedWindowAccess, setHasCheckedWindowAccess] = useState(false);

  const rafId = useRef(0);

  const userRole = localStorage.getItem("userRole");
  const userName = localStorage.getItem("userName") || "User";
  const userEmail = localStorage.getItem("userEmail") || "user@cms.edu";
  const username = localStorage.getItem("username");
  const teacherId = localStorage.getItem("teacher_id") || localStorage.getItem("teacherId");

  const isSidebarExpanded =
    sidebarPinned || isHovering || isPointerDownInSidebar;

  // Check if user has assigned windows (including teachers)
  useEffect(() => {
    const checkUserWindows = async () => {
      try {
        const userIdForWindowCheck = username || teacherId;
        if (!userIdForWindowCheck) {
          setHasAssignedWindows(false);
          setHasCheckedWindowAccess(true);
          return;
        }

        const response = await fetch(
          `${API_BASE_URL}/mark-entry/check-user-windows?user_id=${encodeURIComponent(userIdForWindowCheck)}`,
          {
            method: "GET",
            headers: {
              "Content-Type": "application/json",
            },
          },
        );

        if (response.ok) {
          const data = await response.json();
          let hasWindowAccess = data.has_windows || false;

          // Fallback for teachers: if user-level window mapping is empty,
          // still allow Mark Entry when at least one assigned course has an active window.
          if (!hasWindowAccess && userRole === "teacher" && teacherId) {
            try {
              const teacherCoursesResponse = await fetch(
                `${API_BASE_URL}/teachers/${encodeURIComponent(teacherId)}/courses`,
              );
              if (teacherCoursesResponse.ok) {
                const teacherCourses = await teacherCoursesResponse.json();
                hasWindowAccess = Array.isArray(teacherCourses)
                  ? teacherCourses.some((course) => Boolean(course?.has_window))
                  : false;
              }
            } catch (fallbackError) {
              console.warn("Teacher window fallback check failed:", fallbackError);
            }
          }

          setHasAssignedWindows(hasWindowAccess);
        } else {
          setHasAssignedWindows(false);
        }
      } catch (error) {
        console.error("Error checking user windows:", error);
        setHasAssignedWindows(false);
      } finally {
        setHasCheckedWindowAccess(true);
      }
    };

    if (userRole) {
      checkUserWindows();
    }
  }, [username, teacherId, userRole]);

  const markEntryBaseAllowedRoles = useMemo(
    () =>
      new Set([
        "teacher",
        "user",
        "faculty",
        "staff",
        "coe",
        "curriculum_entry_user",
      ]),
    [],
  );

  const canAccessMarkEntry =
    markEntryBaseAllowedRoles.has(userRole) && hasAssignedWindows;

  const canAccessWindowDetailsRoute = useMemo(
    () => new Set(["admin", "coe", "hod", "curriculum_entry_user"]).has(userRole),
    [userRole],
  );

  useEffect(() => {
    if (!hasCheckedWindowAccess) return;
    const isMarkEntryRoute = location.pathname === "/mark-entry";
    const isWindowDetailsRoute = location.pathname.startsWith("/mark-entry-windows/");

    // Mark entry itself requires user-specific window access.
    if (isMarkEntryRoute && !canAccessMarkEntry) {
      const fallbackPath = userRole === "teacher" ? "/teacher-dashboard" : "/dashboard";
      navigate(fallbackPath, { replace: true });
      return;
    }

    // Window details are also used by COE/Admin from Existing Windows.
    // Keep teacher/user guard while allowing admin-side roles.
    if (isWindowDetailsRoute && !canAccessWindowDetailsRoute && !canAccessMarkEntry) {
      const fallbackPath = userRole === "teacher" ? "/teacher-dashboard" : "/dashboard";
      navigate(fallbackPath, { replace: true });
      return;
    }

  }, [
    canAccessMarkEntry,
    canAccessWindowDetailsRoute,
    hasCheckedWindowAccess,
    location.pathname,
    navigate,
    userRole,
  ]);

  useEffect(() => {
    onExpandedChange?.(isSidebarExpanded);
  }, [isSidebarExpanded, onExpandedChange]);

  useEffect(() => {
    const endInteraction = () => setIsPointerDownInSidebar(false);
    window.addEventListener("pointerup", endInteraction, true);
    window.addEventListener("pointercancel", endInteraction, true);

    return () => {
      window.removeEventListener("pointerup", endInteraction, true);
      window.removeEventListener("pointercancel", endInteraction, true);
    };
  }, []);

  useEffect(() => {
    // If pinned open, ignore hover tracking.
    if (sidebarPinned) {
      setIsHovering(true);
      return;
    }

    const updateHoverFromMouseX = (mouseX) => {
      // Hysteresis:
      // - When collapsed: expand only if mouse is within collapsed width.
      // - When expanded: stay expanded until mouse leaves expanded width.
      setIsHovering((wasHovering) => {
        if (isPointerDownInSidebar) return true;
        if (wasHovering) return mouseX <= EXPANDED_WIDTH_PX;
        return mouseX <= COLLAPSED_WIDTH_PX;
      });
    };

    const onMouseMove = (e) => {
      if (rafId.current) cancelAnimationFrame(rafId.current);
      const mouseX = e.clientX;
      rafId.current = requestAnimationFrame(() =>
        updateHoverFromMouseX(mouseX),
      );
    };

    const onWindowLeave = () => {
      if (isPointerDownInSidebar) return;
      setIsHovering(false);
    };

    window.addEventListener("mousemove", onMouseMove, { passive: true });
    window.addEventListener("mouseleave", onWindowLeave);

    return () => {
      if (rafId.current) cancelAnimationFrame(rafId.current);
      rafId.current = 0;
      window.removeEventListener("mousemove", onMouseMove);
      window.removeEventListener("mouseleave", onWindowLeave);
    };
  }, [sidebarPinned, isPointerDownInSidebar]);

  const menuItems = useMemo(() => {
    const allMenuItems = [
      {
        name: "Dashboard",
        path: userRole === "teacher" ? "/teacher-dashboard" : "/dashboard",
        icon: (
          <svg
            className="w-7 h-7"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"
            />
          </svg>
        ),
        roles: ["admin", "teacher", "coe", "hod", "curriculum_entry_user"],
      },
      {
        name: "Course Dashboard",
        path: "/student/course-dashboard",
        icon: (
          <svg
            className="w-5 h-5"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M3 7h18M3 12h18M3 17h18"
            />
          </svg>
        ),
        roles: ["student"],
      },
      {
        name: "Elective Selection",
        path: "/elective-selection",
        icon: (
          <svg
            className="w-5 h-5"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01"
            />
          </svg>
        ),
        roles: ["student"],
      },
      {
        name: "Elective Excemption",
        path: "/student/elective-exemption",
        icon: (
          <svg
            className="w-5 h-5"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M12 8c-1.657 0-3 1.343-3 3v1H8a2 2 0 00-2 2v3a2 2 0 002 2h8a2 2 0 002-2v-3a2 2 0 00-2-2h-1v-1c0-1.657-1.343-3-3-3zm-1 4v-1a1 1 0 112 0v1h-2zm-6 8h14"
            />
          </svg>
        ),
        roles: ["student"],
      },
      {
        name: "Excemption Status",
        path: "/student/elective-exemption-status",
        icon: (
          <svg
            className="w-5 h-5"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M9 12l2 2 4-4m5 2a9 9 0 11-18 0 9 9 0 0118 0z"
            />
          </svg>
        ),
        roles: ["student"],
      },
      {
        name: "Curriculum",
        path: "/curriculum",
        icon: (
          <svg
            className="w-7 h-7"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
            />
          </svg>
        ),
        roles: ["admin", "hod", "curriculum_entry_user"],
      },
      {
        name: "Users",
        path: "/users",
        icon: (
          <svg
            className="w-7 h-7"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M17 20h5v-1a4 4 0 00-5-3.87M9 20H4v-1a4 4 0 015-3.87m8-8a4 4 0 11-8 0 4 4 0 018 0z"
            />
          </svg>
        ),
        roles: ["admin"],
      },
      {
        name: "Register",
        path: "/student-teacher-dashboard",
        icon: (
          <svg
            className="w-7 h-7"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z"
            />
          </svg>
        ),
        roles: ["admin", "hod"],
      },
      {
        name: "Course Allocation",
        path: "/course-allocation",
        icon: (
          <svg
            className="w-7 h-7"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01"
            />
          </svg>
        ),
        roles: ["admin"],
      },
      {
        name: "Course Selection",
        path: "/teacher/course-selection",
        icon: (
          <svg
            className="w-7 h-7"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01"
            />
          </svg>
        ),
        roles: ["teacher"],
      },
      {
        name: "Teacher Courses",
        path: "/teacher-courses",
        icon: (
          <svg
            className="w-7 h-7"
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
        ),
        roles: ["admin", "teacher"],
      },
      {
        name: "Teacher Assign",
        path: "/admin/teacher-course-assignment",
        icon: (
          <svg
            className="w-7 h-7"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M9 7a3 3 0 116 0 3 3 0 01-6 0M4 19a6 6 0 0112 0M16 8h4m-4 4h4m-4 4h4"
            />
          </svg>
        ),
        roles: ["admin"],
      },
      {
        name: "Mark Entry",
        path: "/mark-entry",
        icon: (
          <svg
            className="w-7 h-7"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
            />
          </svg>
        ),
        roles: [
          "teacher",
          "user",
          "faculty",
          "staff",
          "coe",
          "curriculum_entry_user",
        ],
        requiresWindow: true, // This menu item requires window assignment for non-teachers
      },
      {
        name: "Mark Monitor",
        path: "/mark-monitor",
        icon: (
          <svg
            className="w-7 h-7"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
            />
          </svg>
        ),
        roles: ["admin", "hod"],
      },
      {
        name: "Result Analysis",
        path: "/result-analysis",
        icon: (
          <svg
            className="w-7 h-7"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M11 11V7m4 8v-6m4 10H5a2 2 0 01-2-2V7a2 2 0 012-2h5m4 0h5a2 2 0 012 2v10a2 2 0 01-2 2M9 7h6"
            />
          </svg>
        ),
        roles: ["admin", "hod"],
      },
      {
        name: "My Assigned",
        path: "/my-assigned-students",
        icon: (
          <svg
            className="w-7 h-7"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M17 20h5v-1a4 4 0 00-5-3.87M9 20H4v-1a4 4 0 015-3.87m8-8a4 4 0 11-8 0 4 4 0 018 0z"
            />
          </svg>
        ),
        roles: [
          "user",
          "faculty",
          "staff",
          "curriculum_entry_user",
          "coe",
          "admin",
        ],
      },
      {
        name: "Mark Permissions",
        path: "/mark-entry-permissions",
        icon: (
          <svg
            className="w-7 h-7"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"
            />
          </svg>
        ),
        roles: ["admin", "coe"],
      },
      {
        name: "Honour/Minor Import",
        path: "/hod/honour-minor-eligibility",
        icon: (
          <svg
            className="w-5 h-5"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M12 4v16m8-8H4"
            />
          </svg>
        ),
        roles: ["hod"],
      },
      {
        name: "Exam Absentees",
        path: "/exam-absentees",
        icon: (
          <svg
            className="w-5 h-5"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636"
            />
          </svg>
        ),
        roles: ["coe", "admin"],
      },
      {
        name: "Academic Calendar",
        path: "/academic-calendar",
        icon: (
          <svg
            className="w-7 h-7"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
            />
          </svg>
        ),
        roles: ["admin", "hod"],
      },
      {
        name: "HR Faculty",
        path: "/hr/faculty",
        icon: (
          <svg
            className="w-7 h-7"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M17 20h5v-1a4 4 0 00-5-3.87M9 20H4v-1a4 4 0 015-3.87m8-8a4 4 0 11-8 0 4 4 0 018 0z"
            />
          </svg>
        ),
        roles: ["hr", "admin"],
      },
      {
        name: "HR Appeals",
        path: "/hr/appeals-review",
        icon: (
          <svg
            className="w-7 h-7"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
            />
          </svg>
        ),
        roles: ["hr", "admin"],
      },
      {
        name: "Clusters",
        path: "/clusters",
        icon: (
          <svg
            className="w-7 h-7"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M17 20h5v-2a2 2 0 00-2-2h-3m-6 4H6a2 2 0 01-2-2v-2m16-2V8a2 2 0 00-2-2h-4m-8 0H4a2 2 0 00-2 2v4"
            />
          </svg>
        ),
        roles: ["admin"],
      },
      {
        name: "Sharing",
        path: "/sharing",
        icon: (
          <svg
            className="w-7 h-7"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M8 12a3 3 0 100-6 3 3 0 000 6zm8 8a3 3 0 100-6 3 3 0 000 6zM8 12l8 2"
            />
          </svg>
        ),
        roles: ["admin", "hod"],
      },
      {
        name: "Elective Management",
        path: "/hod/elective-management",
        icon: (
          <svg
            className="w-7 h-7"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4"
            />
          </svg>
        ),
        roles: ["hod"],
      },
      {
        name: "Elective Excemption",
        path: "/hod/elective-exemption",
        icon: (
          <svg
            className="w-7 h-7"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V7a2 2 0 012-2h3l2-2h3l2 2h3a2 2 0 012 2v12a2 2 0 01-2 2z"
            />
          </svg>
        ),
        roles: ["hod"],
      },
    ];

    return allMenuItems.filter((item) => {
      // Mark Entry requires both role permission and assigned windows.
      if (item.name === "Mark Entry") {
        return canAccessMarkEntry;
      }

      // For other items, check if role matches
      if (!item.roles.includes(userRole)) {
        return false;
      }

      return true;
    });
  }, [canAccessMarkEntry, userRole]);

  const isActive = (path) => {
    return (
      location.pathname === path || location.pathname.startsWith(path + "/")
    );
  };

  const handleLogout = () => {
    // Clear all user and teacher related data
    localStorage.removeItem("userId");
    localStorage.removeItem("user_id");
    localStorage.removeItem("userRole");
    localStorage.removeItem("userName");
    localStorage.removeItem("userEmail");
    localStorage.removeItem("username");
    localStorage.removeItem("teacherId");
    localStorage.removeItem("teacher_id");
    localStorage.removeItem("teacher_name");
    localStorage.removeItem("teacher_email");
    localStorage.removeItem("teacher_dept");
    localStorage.removeItem("teacher_designation");
    localStorage.removeItem("faculty_id");
    localStorage.removeItem("token");
    navigate("/");
  };

  return (
    <aside
      onPointerDownCapture={() => setIsPointerDownInSidebar(true)}
      className={`group fixed inset-y-0 left-0 z-50 bg-white border-r border-gray-200 transition-all duration-300 flex flex-col ${
        isSidebarExpanded ? "w-64" : "w-20"
      }`}
    >
      {/* Logo Header */}
      <div
        className={`h-20 flex items-center border-b border-gray-200 transition-all duration-300 ${
          isSidebarExpanded ? "justify-between px-6" : "justify-center"
        }`}
      >
        <div
          className={`flex items-center transition-all duration-300 ${
            isSidebarExpanded ? "space-x-3" : ""
          }`}
        >
          <div className="w-10 h-10 rounded-xl flex items-center justify-center shadow-lg flex-shrink-0 bg-primary">
            <svg
              className="w-6 h-6 text-white"
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
          {isSidebarExpanded && (
            <div className="flex flex-col overflow-hidden">
              <span
                className="text-lg font-bold whitespace-nowrap"
                style={{
                  background: "linear-gradient(to right, #7d53f6, #6c3df0)",
                  WebkitBackgroundClip: "text",
                  WebkitTextFillColor: "transparent",
                }}
              >
                ACADEMICS
              </span>
              <span className="text-xs text-gray-500 font-medium whitespace-nowrap">
                Portal
              </span>
            </div>
          )}
        </div>

        {isSidebarExpanded && (
          <button
            onClick={() => setSidebarPinned((v) => !v)}
            className="p-1.5 rounded-lg hover:bg-gray-100 transition-all duration-300 flex-shrink-0"
            title={sidebarPinned ? "Unpin sidebar" : "Pin sidebar"}
          >
            <svg
              className="w-6 h-6"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d={
                  sidebarPinned
                    ? "M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                    : "M5 5a2 2 0 012-2h10a2 2 0 012 2v16l-7-3.5L5 21V5z"
                }
              />
            </svg>
          </button>
        )}
      </div>

      {/* Navigation */}
      <nav className="flex-1 overflow-y-hidden group-hover:overflow-y-auto px-3 py-3 pr-4 -mr-1 space-y-2">
        {menuItems.map((item) => (
          <button
            key={item.path}
            onClick={() => navigate(item.path)}
            className={`w-full flex items-center rounded-lg transition-all duration-300 ease-in-out ${
              isActive(item.path)
                ? "font-medium"
                : "text-gray-900 hover:bg-gray-200 hover:text-gray-800"
            } ${
              isSidebarExpanded
                ? "px-4 py-2.5 justify-start"
                : "px-0 py-2.5 justify-center"
            }`}
            style={{
              ...(isActive(item.path)
                ? {
                    backgroundColor: "rgb(125, 83, 246)",
                    color: "#ffffff",
                  }
                : {}),
            }}
          >
            <div className="flex items-center gap-3 transition-all duration-300 ease-in-out">
              <div
                className={`flex-shrink-0 w-7 h-7 flex items-center justify-center ${
                  isActive(item.path) ? "text-white" : "text-iconColor"
                }`}
              >
                {item.icon}
              </div>
              {isSidebarExpanded && (
                <span className="whitespace-nowrap overflow-hidden">
                  {item.name}
                </span>
              )}
            </div>
          </button>
        ))}
      </nav>

      {/* User Section */}
      <div className="p-3 border-t border-gray-200 bg-white">
        <div
          className={`flex items-center transition-all duration-300 overflow-hidden ${isSidebarExpanded ? "space-x-3" : "justify-center"}`}
        >
          <div
            className={`w-7 h-7 ms-2 bg-primary rounded-full flex items-center justify-center text-white font-semibold flex-shrink-0`}
          >
            {userName.charAt(0).toUpperCase()}
          </div>
          <div
            className="flex-1 min-w-0 transition-all duration-300"
            style={{
              opacity: isSidebarExpanded ? 1 : 0,
              transform: isSidebarExpanded
                ? "translateX(0)"
                : "translateX(-10px)",
              width: isSidebarExpanded ? "auto" : "0",
            }}
          >
            <p className="text-sm font-medium text-gray-900 truncate">
              {userName}
            </p>
            <p className="text-xs text-gray-500 truncate">{userEmail}</p>
          </div>
        </div>

        <button
          onClick={handleLogout}
          className="mt-3 w-full flex items-center justify-center space-x-2 px-4 py-2 text-sm text-iconColor border hover:bg-primary hover:text-white rounded-lg transition-all duration-300 overflow-hidden"
          style={{
            opacity: isSidebarExpanded ? 1 : 0,
            transform: isSidebarExpanded ? "scaleY(1)" : "scaleY(0)",
            maxHeight: isSidebarExpanded ? "100px" : "0",
            marginTop: isSidebarExpanded ? "0.75rem" : "0",
          }}
        >
          <svg
            className="w-4 h-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"
            />
          </svg>
          <span>Logout</span>
        </button>
      </div>
    </aside>
  );
}

export default React.memo(Sidebar);
