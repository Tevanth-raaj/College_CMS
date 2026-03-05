import React, { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router-dom";
import MainLayout from "../../components/MainLayout";
import { API_BASE_URL } from "../../config";

function ManageCurriculumPage() {
  const { id } = useParams();
  const navigate = useNavigate();

  const [semesters, setSemesters] = useState([]);
  const [honourCards, setHonourCards] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [cardType, setCardType] = useState("normal"); // 'normal' or 'honour'
  const [showDropdown, setShowDropdown] = useState(false);
  const [newSemester, setNewSemester] = useState({
    semester_number: null,
    name: "",
    card_type: "semester",
    vertical_name: "",
  });
  const [newHonourCard, setNewHonourCard] = useState({ title: "" });
  const [showEditModal, setShowEditModal] = useState(false);
  const [editingSemester, setEditingSemester] = useState(null);
  const [editSemesterNumber, setEditSemesterNumber] = useState("");

  useEffect(() => {
    fetchSemesters();
    fetchHonourCards();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  const fetchSemesters = async () => {
    try {
      setLoading(true);
      const response = await fetch(
        `${API_BASE_URL}/curriculum/${id}/semesters`,
      );
      if (!response.ok) {
        throw new Error("Failed to fetch semesters");
      }
      const data = await response.json();

      // Sort semesters by card type and number
      const sortOrder = {
        semester: 1,
        language_elective: 2,
        vertical: 3,
        open_elective: 4,
        one_credit: 5,
      };
      const sorted = (data || []).sort((a, b) => {
        const typeA = sortOrder[a.card_type] || 999;
        const typeB = sortOrder[b.card_type] || 999;
        if (typeA !== typeB) return typeA - typeB;

        // Within same type, sort by number (nulls last)
        if (a.semester_number === null && b.semester_number === null) return 0;
        if (a.semester_number === null) return 1;
        if (b.semester_number === null) return -1;
        return a.semester_number - b.semester_number;
      });

      setSemesters(sorted);
      setError("");
    } catch (err) {
      console.error("Error fetching semesters:", err);
      setError("Failed to load semesters");
    } finally {
      setLoading(false);
    }
  };

  const fetchHonourCards = async () => {
    try {
      const response = await fetch(
        `${API_BASE_URL}/curriculum/${id}/honour-cards`,
      );
      if (!response.ok) {
        throw new Error("Failed to fetch honour cards");
      }
      const data = await response.json();
      setHonourCards(data || []);
    } catch (err) {
      console.error("Error fetching honour cards:", err);
    }
  };

  const handleCreateSemester = async (e) => {
    e.preventDefault();

    // Check for duplicate card type with same number
    if (newSemester.semester_number) {
      const duplicate = semesters.find(
        (sem) =>
          sem.card_type === newSemester.card_type &&
          sem.semester_number === newSemester.semester_number,
      );
      if (duplicate) {
        alert(
          `A ${newSemester.card_type} card with number ${newSemester.semester_number} already exists!`,
        );
        return;
      }
    }

    // Auto-generate name from card type
    const nameMap = {
      semester: "Semester",
      language_elective: "Language Elective",
      vertical: "Vertical",
      open_elective: "Open Elective",
      one_credit: "One Credit Course",
    };
    const generatedName = nameMap[newSemester.card_type] || "Semester";

    try {
      const response = await fetch(
        `${API_BASE_URL}/curriculum/${id}/semester`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            semester_number: newSemester.semester_number,
            name: generatedName,
            card_type: newSemester.card_type,
            vertical_name:
              newSemester.card_type === "vertical" ||
              newSemester.card_type === "open_elective"
                ? newSemester.vertical_name
                : "",
          }),
        },
      );

      if (!response.ok) {
        throw new Error("Failed to create semester");
      }

      setNewSemester({
        semester_number: null,
        name: "",
        card_type: "semester",
        vertical_name: "",
      });
      setShowCreateForm(false);
      setShowDropdown(false);
      fetchSemesters();
    } catch (err) {
      console.error("Error creating semester:", err);
      setError("Failed to create semester");
    }
  };

  const handleCreateHonourCard = async (e) => {
    e.preventDefault();

    try {
      const response = await fetch(
        `${API_BASE_URL}/curriculum/${id}/honour-card`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            title: newHonourCard.title,
          }),
        },
      );

      if (!response.ok) {
        throw new Error("Failed to create honour card");
      }

      setNewHonourCard({ title: "" });
      setShowCreateForm(false);
      setShowDropdown(false);
      fetchHonourCards();
    } catch (err) {
      console.error("Error creating honour card:", err);
      setError("Failed to create honour card");
    }
  };

  const handleAddClick = (type) => {
    setCardType(type);
    setShowCreateForm(true);
    setShowDropdown(false);
  };

  // Helper function to get card styling based on type
  const getCardStyle = (cardType) => {
    const styles = {
      semester: {
        card: "hover:border hover:border-primary",
        gradient: "from-primary to-primary",
        bg: "bg-white",
        badge: "bg-white text-primary border border-primary",
        label: "Semester",
        icon: "📚",
      },
      language_elective: {
        card: "hover:border hover:border-green-700",
        gradient: "from-green-500 to-green-700",
        bg: "bg-green-50",
        badge: "border border-green-700 text-green-700",
        label: "Language Elective",
        icon: "🎯",
      },
      vertical: {
        card: "hover:border hover:border-orange-500",
        gradient: "from-orange-300 to-orange-500",
        bg: "bg-orange-50",
        badge: "bg-orange-100 text-orange-700",
        label: "Vertical",
        icon: "📊",
      },
      open_elective: {
        card: "hover:border hover:border-yellow-500",
        gradient: "from-purple-500 to-purple-700",
        bg: "bg-purple-50",
        badge: "bg-purple-100 text-purple-700",
        label: "Open Elective",
        icon: "🌟",
      },
      one_credit: {
        card: "hover:border hover:border-yellow-500",
        gradient: "from-teal-500 to-teal-700",
        bg: "bg-teal-50",
        badge: "bg-teal-100 text-teal-700",
        label: "One Credit",
        icon: "⭐",
      },
    };
    return styles[cardType] || styles["semester"];
  };

  const handleEditSemester = (semester) => {
    setEditingSemester(semester);
    setEditSemesterNumber(semester.semester_number.toString());
    setShowEditModal(true);
  };

  const handleUpdateSemester = async (e) => {
    e.preventDefault();

    try {
      const response = await fetch(
        `${API_BASE_URL}/semester/${editingSemester.id}`,
        {
          method: "PUT",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            semester_number: parseInt(editSemesterNumber),
          }),
        },
      );

      if (!response.ok) {
        throw new Error("Failed to update semester");
      }

      setShowEditModal(false);
      setEditingSemester(null);
      fetchSemesters();
    } catch (err) {
      console.error("Error updating semester:", err);
      setError("Failed to update semester");
    }
  };

  const handleDeleteSemester = async (semId, semName) => {
    if (
      !window.confirm(
        `Are you sure you want to delete "${semName}"? This action cannot be undone.`,
      )
    ) {
      return;
    }

    try {
      const response = await fetch(`${API_BASE_URL}/semester/${semId}`, {
        method: "DELETE",
      });

      if (!response.ok) {
        throw new Error("Failed to delete semester");
      }

      fetchSemesters();
    } catch (err) {
      console.error("Error deleting semester:", err);
      setError("Failed to delete semester");
    }
  };

  const handleDeleteHonourCard = async (cardId, cardTitle) => {
    if (
      !window.confirm(
        `Are you sure you want to delete honour card "${cardTitle}"? This will also delete all verticals and courses within it. This action cannot be undone.`,
      )
    ) {
      return;
    }

    try {
      const response = await fetch(`${API_BASE_URL}/honour-card/${cardId}`, {
        method: "DELETE",
      });

      if (!response.ok) {
        throw new Error("Failed to delete honour card");
      }

      fetchHonourCards();
    } catch (err) {
      console.error("Error deleting honour card:", err);
      setError("Failed to delete honour card");
    }
  };

  if (loading) {
    return (
      <MainLayout title="Manage Curriculum" subtitle="Loading...">
        <div className="flex justify-center items-center py-20">
          <div className="text-center">
            <svg
              className="animate-spin h-12 w-12 text-blue-600 mx-auto mb-4"
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                className="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                strokeWidth="4"
              ></circle>
              <path
                className="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
            <p className="text-gray-600">Loading curriculum...</p>
          </div>
        </div>
      </MainLayout>
    );
  }

  return (
    <MainLayout
      title="Manage Curriculum"
      subtitle={`Semesters for Curriculum ID: ${id}`}
      actions={
        <div className="flex items-center space-x-3">
          <button
            onClick={() => navigate("/curriculum")}
            className="btn-secondary-custom flex items-center space-x-2"
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
                d="M10 19l-7-7m0 0l7-7m-7 7h18"
              />
            </svg>
            <span>Back</span>
          </button>

          {/* Info Icon with Tooltip */}
          <div className="group relative">
            <div className="w-10 h-10 bg-blue-100 rounded-full flex items-center justify-center cursor-help">
              <span className="text-blue-600 font-bold text-lg">(i)</span>
            </div>
            <div className="invisible group-hover:visible absolute right-0 top-12 w-80 bg-gray-900 text-white text-xs rounded-lg p-4 shadow-xl z-50">
              <div className="mb-2">
                <span className="font-semibold text-blue-300">
                  Normal Card Types:
                </span>
              </div>
              <ul className="list-disc list-inside space-y-1 mb-3 ml-2">
                <li>
                  <span className="font-semibold">Semester:</span> Core semester
                  courses
                </li>
                <li>
                  <span className="font-semibold">Language Elective:</span>{" "}
                  Language elective courses
                </li>
                <li>
                  <span className="font-semibold">Vertical:</span> Specialized
                  verticals
                </li>
                <li>
                  <span className="font-semibold">Open Elective:</span> Open
                  elective courses
                </li>
                <li>
                  <span className="font-semibold">One Credit:</span> One credit
                  courses
                </li>
              </ul>
              <div>
                <span className="font-semibold text-purple-300">
                  Honour Card:
                </span>
                <span className="ml-1">
                  Honour programme with verticals inside
                </span>
              </div>
            </div>
          </div>

          {/* Dropdown Button */}
          <div className="relative">
            <button
              onClick={() => setShowDropdown(!showDropdown)}
              className="btn-primary-custom flex items-center space-x-2"
            >
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
              <span>Add</span>
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
              >
                <path d="M6 9l6 6 6-6" />
              </svg>
            </button>

            {showDropdown && (
              <div className="absolute right-0 mt-2 w-48 bg-white rounded-lg shadow-xl border border-gray-200 z-50">
                <button
                  onClick={() => handleAddClick("normal")}
                  className="w-full text-left px-4 py-3 hover:bg-blue-50 text-gray-700 font-medium rounded-t-lg transition-colors"
                >
                  Normal Card
                </button>
                <button
                  onClick={() => handleAddClick("honour")}
                  className="w-full text-left px-4 py-3 hover:bg-purple-50 text-gray-700 font-medium rounded-b-lg transition-colors"
                >
                  Honour Card
                </button>
              </div>
            )}
          </div>
        </div>
      }
    >
      <div className="space-y-0">
        {/* Error Message */}
        {error && (
          <div className="flex items-start space-x-3 p-4 bg-red-50 border border-red-200 rounded-lg">
            <svg
              className="w-5 h-5 text-red-600 flex-shrink-0 mt-0.5"
              fill="currentColor"
              viewBox="0 0 20 20"
            >
              <path
                fillRule="evenodd"
                d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                clipRule="evenodd"
              />
            </svg>
            <p className="text-sm font-medium text-red-600">{error}</p>
          </div>
        )}

        {/* Create Form */}
        {showCreateForm && (
          <div className="card-custom p-6">
            <h2 className="text-lg font-bold text-gray-900 mb-4">
              {cardType === "normal"
                ? "Add New Normal Card (Semester)"
                : "Add New Honour Card"}
            </h2>
            {cardType === "normal" ? (
              <form
                onSubmit={handleCreateSemester}
                className="flex gap-4 items-end"
              >
                <div className="flex-1">
                  <label className="block text-sm font-semibold text-gray-700 mb-2">
                    Card Type
                  </label>
                  <select
                    value={newSemester.card_type}
                    onChange={(e) =>
                      setNewSemester({
                        ...newSemester,
                        card_type: e.target.value,
                        semester_number: null,
                        vertical_name: "",
                      })
                    }
                    className="input-custom"
                  >
                    <option value="semester">Semester</option>
                    <option value="language_elective">Language Elective</option>
                    <option value="vertical">Vertical</option>
                    <option value="open_elective">Open Elective</option>
                    <option value="one_credit">One Credit Course</option>
                  </select>
                </div>
                {(newSemester.card_type === "semester" ||
                  newSemester.card_type === "vertical") && (
                  <div className="w-32">
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Number
                    </label>
                    <input
                      type="number"
                      value={newSemester.semester_number || ""}
                      onChange={(e) =>
                        setNewSemester({
                          ...newSemester,
                          semester_number: e.target.value
                            ? parseInt(e.target.value)
                            : null,
                        })
                      }
                      placeholder="e.g., 1"
                      min="1"
                      required
                      className="input-custom"
                    />
                  </div>
                )}
                {newSemester.card_type === "vertical" && (
                  <div className="flex-1">
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Vertical Name
                    </label>
                    <input
                      type="text"
                      value={newSemester.vertical_name || ""}
                      onChange={(e) =>
                        setNewSemester({
                          ...newSemester,
                          vertical_name: e.target.value,
                        })
                      }
                      placeholder="e.g., AI & ML"
                      required
                      className="input-custom"
                    />
                  </div>
                )}
                {newSemester.card_type === "open_elective" && (
                  <div className="flex-1">
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Department Type
                    </label>
                    <select
                      value={newSemester.vertical_name || ""}
                      onChange={(e) =>
                        setNewSemester({
                          ...newSemester,
                          vertical_name: e.target.value,
                        })
                      }
                      required
                      className="input-custom"
                    >
                      <option value="">Select Department Type</option>
                      <option value="Same Dept">Same Dept</option>
                      <option value="Different Dept">Different Dept</option>
                    </select>
                  </div>
                )}
                <button type="submit" className="btn-primary-custom">
                  Create Card
                </button>
                <button
                  type="button"
                  onClick={() => setShowCreateForm(false)}
                  className="btn-secondary-custom"
                >
                  Cancel
                </button>
              </form>
            ) : (
              <form
                onSubmit={handleCreateHonourCard}
                className="flex gap-4 items-end"
              >
                <div className="flex-1">
                  <label className="block text-sm font-semibold text-gray-700 mb-2">
                    Honour Card Title
                  </label>
                  <select
                    value={newHonourCard.title}
                    onChange={(e) =>
                      setNewHonourCard({
                        ...newHonourCard,
                        title: e.target.value,
                      })
                    }
                    required
                    className="input-custom"
                  >
                    <option value="">Select Honour Card Type</option>
                    <option value="HONOUR VERTICAL">HONOUR VERTICAL</option>
                    <option value="MINOR VERTICAL">MINOR VERTICAL</option>
                  </select>
                </div>
                <button
                  type="submit"
                  className="bg-primary text-white font-medium px-5 py-2.5 rounded-lg transition-all"
                >
                  Create Honour Card
                </button>
                <button
                  type="button"
                  onClick={() => setShowCreateForm(false)}
                  className="btn-secondary-custom"
                >
                  Cancel
                </button>
              </form>
            )}
          </div>
        )}

        {/* Cards Grid */}
        {semesters.length === 0 && honourCards.length === 0 ? (
          <div className="card-custom p-12 text-center">
            <svg
              className="w-20 h-20 text-gray-300 mx-auto mb-4"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1.5}
                d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
              />
            </svg>
            <h3 className="text-xl font-semibold text-gray-900 mb-2">
              No Cards Yet
            </h3>
            <p className="text-gray-600 mb-6">
              Get started by creating your first card (Normal or Honour)
            </p>
            <button
              onClick={() => setShowDropdown(true)}
              className="btn-primary-custom"
            >
              + Add Card
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-5">
            {/* Normal Cards (All Types) */}
            {semesters.map((sem) => {
              const style = getCardStyle(sem.card_type || "semester");
              const displayLabel =
                sem.card_type === "vertical" && sem.vertical_name
                  ? sem.vertical_name
                  : sem.card_type === "open_elective" && sem.vertical_name
                    ? `Open Elective (${sem.vertical_name})`
                    : style.label;
              return (
                <div
                  key={`sem-${sem.id}`}
                  className={`group card-custom p-6 cursor-pointer transition-all duration-200 relative ${style.bg} ${style.card} border-2 border-transparent hover:border-opacity-50`}
                  onClick={() =>
                    navigate(`/curriculum/${id}/curriculum/semester/${sem.id}`)
                  }
                >
                  {/* Edit Button - Only show for semester and vertical cards */}
                  {(sem.card_type === "semester" ||
                    sem.card_type === "vertical") && (
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        handleEditSemester(sem);
                      }}
                      className="absolute top-3 right-12 w-8 h-8 flex items-center justify-center bg-green-50 text-green-600 rounded-lg hover:bg-green-100 opacity-0 group-hover:opacity-100 transition-all"
                      title="Edit"
                    >
                      <svg
                        width="16"
                        height="16"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="2"
                      >
                        <path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7" />
                        <path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z" />
                      </svg>
                    </button>
                  )}

                  {/* Delete Button */}
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      const cardType = sem.card_type || "semester";
                      const displayName = sem.semester_number
                        ? `${cardType.charAt(0).toUpperCase() + cardType.slice(1)} ${sem.semester_number}`
                        : cardType.charAt(0).toUpperCase() + cardType.slice(1);
                      handleDeleteSemester(sem.id, displayName);
                    }}
                    className="absolute top-3 right-3 w-8 h-8 flex items-center justify-center bg-red-50 text-red-600 rounded-lg hover:bg-red-100 opacity-0 group-hover:opacity-100 transition-all"
                    title="Delete"
                  >
                    <svg
                      width="16"
                      height="16"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="2"
                    >
                      <path d="M3 6h18" />
                      <path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2" />
                      <path d="M10 11v6" />
                      <path d="M14 11v6" />
                    </svg>
                  </button>

                  <div className="text-center">
                    {/* Icon and Number Badge */}
                    <div className="mb-3">
                      {sem.semester_number ? (
                        <div
                          className={`w-16 h-16 bg-gradient-to-br ${style.gradient} rounded-2xl flex items-center justify-center mx-auto group-hover:scale-110 transition-transform shadow-md`}
                        >
                          <span className="text-3xl font-bold text-white">
                            {sem.semester_number}
                          </span>
                        </div>
                      ) : (
                        <div className="text-5xl mb-2">{style.icon}</div>
                      )}
                    </div>

                    {/* Card Name */}
                    <h3 className="text-lg font-bold text-gray-900 mb-2 min-h-[3rem] flex items-center justify-center">
                      {displayLabel}
                    </h3>

                    {/* Card Type Badge */}
                    <span
                      className={`inline-block mb-2 px-3 py-1 ${style.badge} text-xs font-semibold rounded-full`}
                    >
                      {displayLabel}
                    </span>

                    <p className="text-sm text-gray-600">Manage courses →</p>
                  </div>
                </div>
              );
            })}

            {/* Honour Cards */}
            {honourCards.map((card) => (
              <div
                key={`honour-${card.id}`}
                className="relative group card-custom p-6 bg-gradient-to-br from-purple-50 to-pink-50 border-purple-200  hover:border hover:border-primary"
              >
                {/* Delete Button */}
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    handleDeleteHonourCard(card.id, card.title);
                  }}
                  className="absolute top-3 right-3 w-8 h-8 flex items-center justify-center bg-red-50 text-red-600 rounded-lg hover:bg-red-100 opacity-0 group-hover:opacity-100 transition-all"
                  title="Delete"
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
                      d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                    />
                  </svg>
                </button>

                {/* Card Content */}
                <div
                  className="text-center cursor-pointer rounded-lg p-4 transition-all duration-200"
                  onClick={() =>
                    navigate(`/curriculum/${id}/curriculum/honour/${card.id}`)
                  }
                >
                  <div className="text-5xl mb-3">🎖️</div>
                  <h3 className="text-lg font-bold text-gray-900 mb-1">
                    {card.title}
                  </h3>
                  <span className="inline-block mb-2 px-3 py-1 bg-purple-100 text-purple-700 text-xs font-semibold rounded-full">
                    Honour
                  </span>
                  <p className="text-sm text-gray-600">Manage verticals →</p>
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Activity Logs removed - now in Department Overview page */}

        {/* Edit Semester Modal */}
        {showEditModal && editingSemester && (
          <div
            className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 p-4"
            onClick={() => setShowEditModal(false)}
          >
            <div
              className="bg-white rounded-2xl shadow-2xl max-w-md w-full"
              onClick={(e) => e.stopPropagation()}
            >
              <div className="bg-primary text-white px-8 py-5 flex items-center justify-between rounded-t-2xl">
                <div>
                  <h3 className="text-xl font-bold">
                    {editingSemester.card_type === "vertical"
                      ? "Edit Vertical"
                      : "Edit Semester"}
                  </h3>
                  <p className="text-sm text-green-100">
                    {editingSemester.card_type === "vertical"
                      ? "Update vertical number"
                      : "Update semester number"}
                  </p>
                </div>
                <button
                  onClick={() => setShowEditModal(false)}
                  className="text-white hover:bg-white/20 rounded-lg p-2 transition-all"
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
                      d="M6 18L18 6M6 6l12 12"
                    />
                  </svg>
                </button>
              </div>

              <form onSubmit={handleUpdateSemester} className="p-8 space-y-5">
                <div>
                  <label className="block text-sm font-semibold text-gray-700 mb-2">
                    {editingSemester.card_type === "vertical"
                      ? "Vertical Number"
                      : "Semester Number"}
                  </label>
                  <input
                    type="number"
                    value={editSemesterNumber}
                    onChange={(e) => setEditSemesterNumber(e.target.value)}
                    placeholder="e.g., 1"
                    required
                    min="1"
                    className="input-custom"
                  />
                </div>

                <div className="flex gap-3 justify-end pt-2">
                  <button
                    type="button"
                    onClick={() => setShowEditModal(false)}
                    className="btn-secondary-custom"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="btn-primary-custom text-white font-medium px-5 py-2.5 rounded-lg transition-all duration-200 shadow-sm hover:shadow-md active:scale-95"
                  >
                    {editingSemester.card_type === "vertical"
                      ? "Update Vertical"
                      : "Update Semester"}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}
      </div>
    </MainLayout>
  );
}

export default ManageCurriculumPage;
