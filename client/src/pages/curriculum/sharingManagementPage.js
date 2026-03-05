import React, { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import MainLayout from "../../components/MainLayout";
import { API_BASE_URL } from "../../config";

function SharingManagementPage() {
  const navigate = useNavigate();
  const [clusters, setClusters] = useState([]);
  const [selectedCluster, setSelectedCluster] = useState(null);
  const [clusterDepartments, setClusterDepartments] = useState([]);
  const [selectedCurriculum, setSelectedCurriculum] = useState(null);
  const [sharingInfo, setSharingInfo] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [clusterContent, setClusterContent] = useState(null);
  const [showDepartmentSelector, setShowDepartmentSelector] = useState(false);
  const [currentSharingItem, setCurrentSharingItem] = useState(null);
  const [selectedDepartments, setSelectedDepartments] = useState([]);
  const [sharingMode, setSharingMode] = useState("replace"); // 'replace', 'add', or 'remove'
  const [sharedWithDepartments, setSharedWithDepartments] = useState([]); // Departments that currently have the item

  useEffect(() => {
    fetchClusters();
  }, []);

  const fetchClusters = async () => {
    try {
      setLoading(true);
      const response = await fetch(`${API_BASE_URL}/clusters`);
      if (!response.ok) {
        throw new Error("Failed to fetch clusters");
      }
      const data = await response.json();
      setClusters(data || []);
      setError("");
    } catch (err) {
      console.error("Error fetching clusters:", err);
      setError("Failed to load clusters");
    } finally {
      setLoading(false);
    }
  };

  const fetchClusterDepartments = async (clusterId) => {
    try {
      const response = await fetch(
        `${API_BASE_URL}/cluster/${clusterId}/departments`,
      );
      if (!response.ok) {
        throw new Error("Failed to fetch cluster departments");
      }
      const data = await response.json();
      setClusterDepartments(data || []);
    } catch (err) {
      console.error("Error fetching cluster departments:", err);
      setClusterDepartments([]);
    }
  };

  const handleSelectCluster = (cluster) => {
    setSelectedCluster(cluster);
    setSelectedCurriculum(null);
    setSharingInfo(null);
    fetchClusterDepartments(cluster.id);
    fetchClusterContent(cluster.id);
  };

  const fetchSharingInfo = async (curriculumId) => {
    try {
      const response = await fetch(
        `${API_BASE_URL}/curriculum/${curriculumId}/sharing`,
      );
      if (!response.ok) {
        throw new Error("Failed to fetch sharing info");
      }
      const data = await response.json();
      setSharingInfo(data);
    } catch (err) {
      console.error("Error fetching sharing info:", err);
      setError("Failed to load sharing information");
    }
  };

  const fetchClusterContent = async (clusterId) => {
    try {
      const response = await fetch(
        `${API_BASE_URL}/cluster/${clusterId}/shared-content`,
      );
      if (!response.ok) {
        throw new Error("Failed to fetch cluster content");
      }
      const data = await response.json();
      setClusterContent(data);
    } catch (err) {
      console.error("Error fetching cluster content:", err);
    }
  };

  const handleSelectCurriculum = (dept) => {
    setSelectedCurriculum(dept);
    fetchSharingInfo(dept.curriculum_id);
  };

  const handleToggleVisibility = async (
    itemType,
    itemId,
    currentVisibility,
    targetDepartments = null,
    mode = "replace",
  ) => {
    // When modifying (add/remove modes), keep the current visibility
    // Only toggle when it's a replace mode without current sharing (new share or unshare)
    let newVisibility;
    if (mode === "add" || mode === "remove") {
      // For add/remove modes, maintain current visibility (should be CLUSTER)
      newVisibility = currentVisibility;
    } else {
      // For replace mode or toggle, flip the visibility
      newVisibility = currentVisibility === "UNIQUE" ? "CLUSTER" : "UNIQUE";
    }

    // If changing to CLUSTER and no departments specified, show selector
    if (
      newVisibility === "CLUSTER" &&
      !targetDepartments &&
      sharingInfo?.cluster_departments?.length > 0
    ) {
      setCurrentSharingItem({ itemType, itemId, currentVisibility });
      setSelectedDepartments([]);
      setSharingMode("replace"); // Default mode for new sharing
      setShowDepartmentSelector(true);
      return;
    }

    try {
      const requestBody = {
        item_type: itemType,
        item_id: itemId,
        visibility: newVisibility,
        sharing_mode: mode || "replace",
      };

      // Add target departments if specified
      if (targetDepartments && targetDepartments.length > 0) {
        requestBody.target_departments = targetDepartments;
      }

      const response = await fetch(`${API_BASE_URL}/sharing/visibility`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(requestBody),
      });

      if (!response.ok) {
        throw new Error("Failed to update visibility");
      }

      setSuccess(
        `Updated to ${newVisibility === "CLUSTER" ? "Shared" : "Private"}`,
      );
      setTimeout(() => setSuccess(""), 3000);

      // Refresh sharing info and cluster content
      if (selectedCurriculum) {
        await fetchSharingInfo(selectedCurriculum.curriculum_id);
      }
      if (selectedCluster) {
        await fetchClusterContent(selectedCluster.id);
      }
    } catch (err) {
      console.error("Error updating visibility:", err);
      setError("Failed to update sharing settings");
    }
  };

  const handleConfirmSharing = async () => {
    if (currentSharingItem) {
      // Close modal immediately for responsive UI
      setShowDepartmentSelector(false);
      const itemToShare = currentSharingItem;
      const depsToShare =
        selectedDepartments.length > 0 ? selectedDepartments : null;
      const mode = sharingMode;

      setCurrentSharingItem(null);
      setSelectedDepartments([]);
      setSharedWithDepartments([]);

      // Continue operation in background
      handleToggleVisibility(
        itemToShare.itemType,
        itemToShare.itemId,
        itemToShare.currentVisibility,
        depsToShare,
        mode,
      );
    }
  };

  const handleModifySharing = async (itemType, itemId) => {
    setCurrentSharingItem({ itemType, itemId, currentVisibility: "CLUSTER" });
    setSelectedDepartments([]);
    setSharingMode("add"); // Default to add mode for modifications

    // Fetch the list of departments that currently have this item
    try {
      const response = await fetch(
        `${API_BASE_URL}/sharing/${itemType}/${itemId}/departments`,
      );
      if (response.ok) {
        const data = await response.json();
        setSharedWithDepartments(data.shared_with || []);
      } else {
        setSharedWithDepartments([]);
      }
    } catch (err) {
      console.error("Error fetching shared departments:", err);
      setSharedWithDepartments([]);
    }

    setShowDepartmentSelector(true);
  };

  const toggleDepartmentSelection = (deptId) => {
    setSelectedDepartments((prev) => {
      if (prev.includes(deptId)) {
        return prev.filter((id) => id !== deptId);
      } else {
        return [...prev, deptId];
      }
    });
  };

  const selectAllDepartments = () => {
    if (sharingInfo?.cluster_departments) {
      // Filter departments based on sharing mode before selecting all
      const filteredDepts = sharingInfo.cluster_departments.filter((dept) => {
        const isSharedWith = sharedWithDepartments.includes(dept.department_id);

        if (sharingMode === "add") {
          return !isSharedWith;
        } else if (sharingMode === "remove") {
          return isSharedWith;
        }
        return true;
      });

      setSelectedDepartments(filteredDepts.map((d) => d.department_id));
    }
  };

  const deselectAllDepartments = () => {
    setSelectedDepartments([]);
  };

  const renderDepartmentSelector = () => {
    if (!showDepartmentSelector || !sharingInfo?.cluster_departments)
      return null;

    const isModifying = currentSharingItem?.currentVisibility === "CLUSTER";

    return (
      <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
        <div className="bg-white rounded-lg p-6 max-w-md w-full mx-4">
          <h3 className="text-lg font-bold text-gray-900 mb-4">
            {isModifying
              ? "Modify Sharing List"
              : "Select Departments to Share With"}
          </h3>

          {/* Sharing Mode Selector - only show when modifying */}
          {isModifying && (
            <div className="mb-4 p-3 bg-blue-50 rounded-lg">
              <p className="text-xs font-semibold text-gray-700 mb-2">
                Sharing Mode:
              </p>
              <div className="space-y-2">
                <label className="flex items-center cursor-pointer">
                  <input
                    type="radio"
                    name="sharingMode"
                    value="add"
                    checked={sharingMode === "add"}
                    onChange={(e) => setSharingMode(e.target.value)}
                    className="mr-2 h-4 w-4 text-blue-600"
                  />
                  <span className="text-sm text-gray-900">
                    <strong>Add</strong> - Add selected departments to current
                    sharing list
                  </span>
                </label>
                <label className="flex items-center cursor-pointer">
                  <input
                    type="radio"
                    name="sharingMode"
                    value="remove"
                    checked={sharingMode === "remove"}
                    onChange={(e) => setSharingMode(e.target.value)}
                    className="mr-2 h-4 w-4 text-blue-600"
                  />
                  <span className="text-sm text-gray-900">
                    <strong>Remove</strong> - Remove selected departments from
                    sharing
                  </span>
                </label>
                <label className="flex items-center cursor-pointer">
                  <input
                    type="radio"
                    name="sharingMode"
                    value="replace"
                    checked={sharingMode === "replace"}
                    onChange={(e) => setSharingMode(e.target.value)}
                    className="mr-2 h-4 w-4 text-blue-600"
                  />
                  <span className="text-sm text-gray-900">
                    <strong>Replace</strong> - Replace entire sharing list with
                    selected
                  </span>
                </label>
              </div>
            </div>
          )}

          <div className="mb-4">
            <div className="flex justify-between items-center mb-2">
              <p className="text-sm text-gray-600">Choose departments:</p>
              <div className="space-x-2">
                <button
                  onClick={selectAllDepartments}
                  className="text-xs text-blue-600 hover:text-blue-700"
                >
                  Select All
                </button>
                <button
                  onClick={deselectAllDepartments}
                  className="text-xs text-gray-600 hover:text-gray-700"
                >
                  Clear
                </button>
              </div>
            </div>

            <div className="space-y-2 max-h-64 overflow-y-auto">
              {(() => {
                // First filter by curriculum template compatibility
                const compatibleDepts = sharingInfo.cluster_departments.filter(
                  (dept) =>
                    dept.curriculum_template ===
                    sharingInfo.curriculum_template,
                );

                // Then filter based on sharing mode
                const filteredDepts = compatibleDepts.filter((dept) => {
                  const isSharedWith = sharedWithDepartments.includes(
                    dept.department_id,
                  );

                  if (sharingMode === "add") {
                    return !isSharedWith;
                  } else if (sharingMode === "remove") {
                    return isSharedWith;
                  }
                  return true;
                });

                if (compatibleDepts.length === 0) {
                  return (
                    <div className="text-center py-4 text-gray-500 text-sm">
                      <div className="mb-2">
                        <svg
                          className="w-12 h-12 text-amber-400 mx-auto mb-2"
                          fill="none"
                          stroke="currentColor"
                          viewBox="0 0 24 24"
                        >
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth={2}
                            d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
                          />
                        </svg>
                      </div>
                      <strong className="block mb-1">
                        No compatible departments
                      </strong>
                      <p className="text-xs">
                        All other departments use a different curriculum
                        template (
                        {sharingInfo.curriculum_template === "2022"
                          ? "2026"
                          : "2022"}
                        ).
                      </p>
                      <p className="text-xs mt-1">
                        Only departments with template{" "}
                        <strong>{sharingInfo.curriculum_template}</strong> can
                        receive this content.
                      </p>
                    </div>
                  );
                }

                if (filteredDepts.length === 0) {
                  return (
                    <div className="text-center py-4 text-gray-500 text-sm">
                      {sharingMode === "add"
                        ? "All compatible departments already have this item"
                        : sharingMode === "remove"
                          ? "No compatible departments have this item yet"
                          : "No compatible departments available"}
                    </div>
                  );
                }

                return filteredDepts.map((dept) => (
                  <label
                    key={dept.department_id}
                    className="flex items-center justify-between p-2 hover:bg-gray-50 rounded cursor-pointer"
                  >
                    <div className="flex items-center flex-1">
                      <input
                        type="checkbox"
                        checked={selectedDepartments.includes(
                          dept.department_id,
                        )}
                        onChange={() =>
                          toggleDepartmentSelection(dept.department_id)
                        }
                        className="mr-3 h-4 w-4 text-blue-600"
                      />
                      <span className="text-sm text-gray-900">{dept.name}</span>
                    </div>
                    <span
                      className={`text-xs px-2 py-0.5 rounded ${
                        dept.curriculum_template === "2022"
                          ? "bg-amber-100 text-amber-800"
                          : "bg-emerald-100 text-emerald-800"
                      }`}
                    >
                      {dept.curriculum_template}
                    </span>
                  </label>
                ));
              })()}
            </div>
          </div>

          <div className="text-xs text-gray-500 mb-4">
            {selectedDepartments.length === 0
              ? "Share with all departments by default"
              : `Sharing with ${selectedDepartments.length} selected department(s)`}
          </div>

          <div className="flex justify-end space-x-3">
            <button
              onClick={() => {
                setShowDepartmentSelector(false);
                setCurrentSharingItem(null);
                setSelectedDepartments([]);
                setSharedWithDepartments([]);
              }}
              className="px-4 py-2 text-sm text-gray-700 bg-gray-100 hover:bg-gray-200 rounded-lg"
            >
              Cancel
            </button>
            <button
              onClick={handleConfirmSharing}
              className="px-4 py-2 text-sm text-white bg-blue-600 hover:bg-blue-700 rounded-lg"
            >
              {sharingMode === "add"
                ? "Add"
                : sharingMode === "remove"
                  ? "Remove"
                  : "Share"}
            </button>
          </div>
        </div>
      </div>
    );
  };

  const renderItemList = (items, itemType, title) => {
    if (!items || items.length === 0) {
      return (
        <div className="text-center py-8 text-gray-500">
          <p className="text-sm">No {title.toLowerCase()} items</p>
        </div>
      );
    }

    return (
      <div className="space-y-2">
        {items.map((item, index) => {
          const isOwner = item.is_owner !== false; // Default to true if not specified
          const isShared = item.visibility === "CLUSTER";
          const isReceived = isShared && !isOwner;

          return (
            <div
              key={item.id}
              className="flex items-start justify-between p-3 bg-gray-50 rounded-lg border border-gray-200 hover:border-gray-300 transition-colors"
            >
              <div className="flex-1 flex items-start space-x-3">
                <span className="text-sm font-semibold text-gray-500 mt-0.5">
                  {index + 1}.
                </span>
                <p className="text-sm text-gray-900 flex-1">{item.text}</p>
              </div>

              <div className="flex items-center space-x-2">
                {/* Modify button - only show for owned shared items */}
                {isOwner && isShared && (
                  <button
                    onClick={() => handleModifySharing(itemType, item.id)}
                    className="px-3 py-1.5 rounded-lg text-xs font-semibold bg-blue-100 text-blue-700 hover:bg-blue-200 transition-all"
                    title="Modify sharing list (add/remove departments)"
                  >
                    Modify
                  </button>
                )}

                {/* Toggle visibility button */}
                <button
                  onClick={() =>
                    isOwner &&
                    handleToggleVisibility(itemType, item.id, item.visibility)
                  }
                  disabled={!isOwner}
                  className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition-all ${
                    isReceived
                      ? "bg-blue-100 text-blue-700 cursor-not-allowed"
                      : isShared
                        ? "bg-green-100 text-green-700 hover:bg-green-200 cursor-pointer"
                        : "bg-gray-100 text-gray-600 hover:bg-gray-200 cursor-pointer"
                  } ${!isOwner ? "opacity-75" : ""}`}
                  title={
                    isReceived
                      ? "Received from another department (read-only)"
                      : isShared
                        ? "Shared with cluster - click to unshare"
                        : "Private - click to share with cluster"
                  }
                >
                  {isReceived ? (
                    <div className="flex items-center space-x-1">
                      <svg
                        className="w-3 h-3"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M7 16V4m0 0L3 8m4-4l4 4m6 0v12m0 0l4-4m-4 4l-4-4"
                        />
                      </svg>
                      <span>Received</span>
                    </div>
                  ) : isShared ? (
                    <div className="flex items-center space-x-1">
                      <svg
                        className="w-3 h-3"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"
                        />
                      </svg>
                      <span>Shared</span>
                    </div>
                  ) : (
                    <div className="flex items-center space-x-1">
                      <svg
                        className="w-3 h-3"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
                        />
                      </svg>
                      <span>Private</span>
                    </div>
                  )}
                </button>
              </div>
            </div>
          );
        })}
      </div>
    );
  };

  const getCardDisplayName = (card) => {
    const cardType = card.card_type || "semester";
    if (cardType === "semester" && card.semester_number)
      return `Semester ${card.semester_number}`;
    if (cardType === "vertical") {
      return card.semester_number
        ? `Vertical ${card.semester_number}`
        : "Vertical";
    }
    if (cardType === "language_elective") return "Language Elective";
    if (cardType === "open_elective") {
      const vn = card.vertical_name || "";
      if (vn === "Same Dept") return "Open Elective (Same Dept)";
      if (vn === "Different Dept") return "Open Elective (Diff Dept)";
      return "Open Elective";
    }
    if (cardType === "one_credit") return "One Credit";
    return `Card ${card.id}`;
  };

  const getCardIcon = (cardType) => {
    switch (cardType) {
      case "vertical":
        return (
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
          />
        );
      case "language_elective":
      case "open_elective":
        return (
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"
          />
        );
      default:
        return (
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
          />
        );
    }
  };

  const renderNormalCardsList = () => {
    if (
      !sharingInfo ||
      !sharingInfo.normal_cards ||
      sharingInfo.normal_cards.length === 0
    ) {
      return (
        <div className="card-custom p-6">
          <h3 className="text-md font-bold text-gray-900 mb-4">
            Curriculum Cards (Semesters, Verticals, Language Electives)
          </h3>
          <div className="text-center py-8 text-gray-500">
            <p className="text-sm">No curriculum cards available</p>
          </div>
        </div>
      );
    }

    return (
      <div className="card-custom p-6">
        <h3 className="text-md font-bold text-gray-900 mb-4">
          Curriculum Cards (Semesters, Verticals, Language Electives)
        </h3>
        <div className="space-y-4">
          {sharingInfo.normal_cards.map((card, index) => {
            const cardIsOwner = card.is_owner !== false;
            const cardIsShared = card.visibility === "CLUSTER";
            const cardIsReceived = cardIsShared && !cardIsOwner;

            return (
              <div
                key={card.id}
                className="border border-gray-200 rounded-lg overflow-hidden"
              >
                {/* Card Header */}
                <div className="flex items-center justify-between p-3 bg-gray-50">
                  <div className="flex items-center space-x-3">
                    <svg
                      className="w-5 h-5 text-gray-600"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      {getCardIcon(card.card_type)}
                    </svg>
                    <span className="font-semibold text-gray-900">
                      {getCardDisplayName(card)}
                    </span>
                    {card.courses && (
                      <span className="text-xs text-gray-500">
                        ({card.courses.length} courses)
                      </span>
                    )}
                  </div>

                  <div className="flex items-center space-x-2">
                    {/* Modify button for cards */}
                    {cardIsOwner && cardIsShared && (
                      <button
                        onClick={() => handleModifySharing("semester", card.id)}
                        className="px-3 py-1.5 rounded-lg text-xs font-semibold bg-blue-100 text-blue-700 hover:bg-blue-200 transition-all"
                        title="Modify sharing list (add/remove departments)"
                      >
                        Modify
                      </button>
                    )}

                    {/* Toggle visibility button */}
                    <button
                      onClick={() =>
                        cardIsOwner &&
                        handleToggleVisibility(
                          "semester",
                          card.id,
                          card.visibility,
                        )
                      }
                      disabled={!cardIsOwner}
                      className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition-all ${
                        cardIsReceived
                          ? "bg-blue-100 text-blue-700 cursor-not-allowed"
                          : cardIsShared
                            ? "bg-green-100 text-green-700 hover:bg-green-200 cursor-pointer"
                            : "bg-gray-100 text-gray-600 hover:bg-gray-200 cursor-pointer"
                      } ${!cardIsOwner ? "opacity-75" : ""}`}
                      title={
                        cardIsReceived
                          ? "Received from another department (read-only)"
                          : cardIsShared
                            ? "Shared with cluster - click to unshare"
                            : "Private - click to share with cluster"
                      }
                    >
                      {cardIsReceived ? (
                        <div className="flex items-center space-x-1">
                          <svg
                            className="w-3 h-3"
                            fill="none"
                            stroke="currentColor"
                            viewBox="0 0 24 24"
                          >
                            <path
                              strokeLinecap="round"
                              strokeLinejoin="round"
                              strokeWidth={2}
                              d="M7 16V4m0 0L3 8m4-4l4 4m6 0v12m0 0l4-4m-4 4l-4-4"
                            />
                          </svg>
                          <span>Received</span>
                        </div>
                      ) : cardIsShared ? (
                        <div className="flex items-center space-x-1">
                          <svg
                            className="w-3 h-3"
                            fill="none"
                            stroke="currentColor"
                            viewBox="0 0 24 24"
                          >
                            <path
                              strokeLinecap="round"
                              strokeLinejoin="round"
                              strokeWidth={2}
                              d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"
                            />
                          </svg>
                          <span>Shared</span>
                        </div>
                      ) : (
                        <div className="flex items-center space-x-1">
                          <svg
                            className="w-3 h-3"
                            fill="none"
                            stroke="currentColor"
                            viewBox="0 0 24 24"
                          >
                            <path
                              strokeLinecap="round"
                              strokeLinejoin="round"
                              strokeWidth={2}
                              d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
                            />
                          </svg>
                          <span>Private</span>
                        </div>
                      )}
                    </button>
                  </div>
                </div>

                {/* Courses List */}
                {card.courses && card.courses.length > 0 && (
                  <div className="p-3 space-y-2">
                    {card.courses.map((course) => {
                      const courseIsOwner = course.is_owner !== false;
                      const courseIsShared = course.visibility === "CLUSTER";
                      const courseIsReceived = courseIsShared && !courseIsOwner;

                      return (
                        <div
                          key={course.id}
                          className="flex items-center justify-between p-2 bg-white rounded border border-gray-100"
                        >
                          <div className="flex-1">
                            <p className="text-sm font-medium text-gray-900">
                              {course.course_code}
                            </p>
                            <p className="text-xs text-gray-600">
                              {course.course_name}
                            </p>
                          </div>
                          <div className="flex items-center space-x-2">
                            {/* Modify button for courses */}
                            {courseIsOwner && courseIsShared && (
                              <button
                                onClick={() =>
                                  handleModifySharing("course", course.id)
                                }
                                className="px-2 py-1 rounded text-xs font-semibold bg-blue-100 text-blue-700 hover:bg-blue-200 transition-all"
                                title="Modify sharing list (add/remove departments)"
                              >
                                Modify
                              </button>
                            )}
                            {/* Toggle visibility button */}
                            <button
                              onClick={() =>
                                courseIsOwner &&
                                handleToggleVisibility(
                                  "course",
                                  course.id,
                                  course.visibility,
                                )
                              }
                              disabled={!courseIsOwner}
                              className={`ml-3 px-2 py-1 rounded text-xs font-semibold transition-all ${
                                courseIsReceived
                                  ? "bg-blue-100 text-blue-700 cursor-not-allowed opacity-75"
                                  : courseIsShared
                                    ? "bg-green-100 text-green-700 hover:bg-green-200 cursor-pointer"
                                    : "bg-gray-100 text-gray-600 hover:bg-gray-200 cursor-pointer"
                              }`}
                              title={
                                courseIsReceived
                                  ? "Received (read-only)"
                                  : courseIsShared
                                    ? "Shared - click to unshare"
                                    : "Private - click to share"
                              }
                            >
                              {courseIsReceived
                                ? "Received"
                                : courseIsShared
                                  ? "Shared"
                                  : "Private"}
                            </button>{" "}
                          </div>{" "}
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </div>
    );
  };

  const renderHonourCardsList = () => {
    if (
      !sharingInfo ||
      !sharingInfo.honour_cards ||
      sharingInfo.honour_cards.length === 0
    ) {
      return null; // Don't show section if no honour cards
    }

    return (
      <div className="card-custom p-6">
        <h3 className="text-md font-bold text-gray-900 mb-4">
          Honour Programme Cards
        </h3>
        <div className="space-y-3">
          {sharingInfo.honour_cards.map((honourCard) => {
            const cardIsOwner = honourCard.is_owner !== false;
            const cardIsShared = honourCard.visibility === "CLUSTER";
            const cardIsReceived = cardIsShared && !cardIsOwner;

            return (
              <div
                key={honourCard.id}
                className="border border-purple-200 rounded-lg overflow-hidden bg-purple-50"
              >
                <div className="flex items-center justify-between p-3">
                  <div className="flex items-center space-x-3">
                    <svg
                      className="w-5 h-5 text-purple-600"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z"
                      />
                    </svg>
                    <div>
                      <span className="font-semibold text-gray-900">
                        {honourCard.title}
                      </span>
                      {honourCard.semester_number && (
                        <span className="ml-2 text-xs text-gray-500">
                          (Semester {honourCard.semester_number})
                        </span>
                      )}
                    </div>
                  </div>

                  <div className="flex items-center space-x-2">
                    {cardIsOwner && cardIsShared && (
                      <button
                        onClick={() =>
                          handleModifySharing("honour_card", honourCard.id)
                        }
                        className="px-3 py-1.5 rounded-lg text-xs font-semibold bg-purple-200 text-purple-800 hover:bg-purple-300 transition-all"
                        title="Modify sharing list (add/remove departments)"
                      >
                        Modify
                      </button>
                    )}

                    <button
                      onClick={() =>
                        cardIsOwner &&
                        handleToggleVisibility(
                          "honour_card",
                          honourCard.id,
                          honourCard.visibility,
                        )
                      }
                      disabled={!cardIsOwner}
                      className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition-all ${
                        cardIsReceived
                          ? "bg-blue-100 text-blue-700 cursor-not-allowed"
                          : cardIsShared
                            ? "bg-green-100 text-green-700 hover:bg-green-200 cursor-pointer"
                            : "bg-gray-100 text-gray-600 hover:bg-gray-200 cursor-pointer"
                      } ${!cardIsOwner ? "opacity-75" : ""}`}
                      title={
                        cardIsReceived
                          ? "Received from another department (read-only)"
                          : cardIsShared
                            ? "Shared with cluster - click to unshare"
                            : "Private - click to share with cluster"
                      }
                    >
                      {cardIsReceived ? (
                        <div className="flex items-center space-x-1">
                          <svg
                            className="w-3 h-3"
                            fill="none"
                            stroke="currentColor"
                            viewBox="0 0 24 24"
                          >
                            <path
                              strokeLinecap="round"
                              strokeLinejoin="round"
                              strokeWidth={2}
                              d="M7 16V4m0 0L3 8m4-4l4 4m6 0v12m0 0l4-4m-4 4l-4-4"
                            />
                          </svg>
                          <span>Received</span>
                        </div>
                      ) : cardIsShared ? (
                        <div className="flex items-center space-x-1">
                          <svg
                            className="w-3 h-3"
                            fill="none"
                            stroke="currentColor"
                            viewBox="0 0 24 24"
                          >
                            <path
                              strokeLinecap="round"
                              strokeLinejoin="round"
                              strokeWidth={2}
                              d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"
                            />
                          </svg>
                          <span>Shared</span>
                        </div>
                      ) : (
                        <div className="flex items-center space-x-1">
                          <svg
                            className="w-3 h-3"
                            fill="none"
                            stroke="currentColor"
                            viewBox="0 0 24 24"
                          >
                            <path
                              strokeLinecap="round"
                              strokeLinejoin="round"
                              strokeWidth={2}
                              d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
                            />
                          </svg>
                          <span>Private</span>
                        </div>
                      )}
                    </button>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    );
  };

  const renderClusterSharedContent = () => {
    if (!clusterContent || !clusterContent.departments) return null;

    return (
      <div className="card-custom p-6 mt-6">
        <h2 className="text-lg font-bold text-gray-900 mb-5 flex items-center space-x-2">
          <svg
            className="w-5 h-5 text-green-600"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"
            />
          </svg>
          <span>Shared Content in Cluster</span>
        </h2>

        {clusterContent.departments.map((dept) => {
          const hasSharedContent =
            (dept.mission && dept.mission.length > 0) ||
            (dept.peos && dept.peos.length > 0) ||
            (dept.psos && dept.psos.length > 0) ||
            (dept.semesters && dept.semesters.length > 0) ||
            (dept.honour_cards && dept.honour_cards.length > 0);

          if (!hasSharedContent) return null;

          return (
            <div
              key={dept.department_id}
              className="mb-6 p-4 bg-green-50 rounded-lg border border-green-200"
            >
              <h3 className="font-semibold text-gray-900 mb-3">{dept.name}</h3>

              {/* Department Data */}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
                {dept.mission && dept.mission.length > 0 && (
                  <div>
                    <p className="text-xs font-semibold text-gray-600 mb-2">
                      Mission
                    </p>
                    <ul className="space-y-1">
                      {dept.mission.map((item) => (
                        <li key={item.id} className="text-sm text-gray-700">
                          • {item.text}
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
                {dept.peos && dept.peos.length > 0 && (
                  <div>
                    <p className="text-xs font-semibold text-gray-600 mb-2">
                      PEOs
                    </p>
                    <ul className="space-y-1">
                      {dept.peos.map((item) => (
                        <li key={item.id} className="text-sm text-gray-700">
                          • {item.text}
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
                {dept.psos && dept.psos.length > 0 && (
                  <div>
                    <p className="text-xs font-semibold text-gray-600 mb-2">
                      PSOs
                    </p>
                    <ul className="space-y-1">
                      {dept.psos.map((item) => (
                        <li key={item.id} className="text-sm text-gray-700">
                          • {item.text}
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>

              {/* Shared Semesters */}
              {dept.semesters && dept.semesters.length > 0 && (
                <div className="mt-4">
                  <p className="text-xs font-semibold text-gray-600 mb-2">
                    Shared Curriculum Cards
                  </p>
                  <div className="space-y-2">
                    {dept.semesters.map((card) => {
                      const getDisplayName = () => {
                        const cardType = card.card_type || "semester";
                        if (cardType === "semester" && card.semester_number)
                          return `Semester ${card.semester_number}`;
                        if (cardType === "vertical")
                          return card.semester_number
                            ? `Vertical ${card.semester_number}`
                            : "Vertical";
                        if (cardType === "language_elective")
                          return "Language Elective";
                        if (cardType === "open_elective") {
                          const vn = card.vertical_name || "";
                          if (vn === "Same Dept")
                            return "Open Elective (Same Dept)";
                          if (vn === "Different Dept")
                            return "Open Elective (Diff Dept)";
                          return "Open Elective";
                        }
                        if (cardType === "one_credit") return "One Credit";
                        return `Card ${card.id}`;
                      };
                      return (
                        <div
                          key={card.id}
                          className="bg-white rounded p-3 border border-green-200"
                        >
                          <p className="text-sm font-semibold text-gray-900 mb-2">
                            {getDisplayName()}
                          </p>
                          {card.courses && card.courses.length > 0 && (
                            <div className="space-y-1">
                              {card.courses.map((course) => (
                                <div
                                  key={course.id}
                                  className="text-xs text-gray-700 pl-3"
                                >
                                  • {course.course_code} - {course.course_name}
                                </div>
                              ))}
                            </div>
                          )}
                        </div>
                      );
                    })}
                  </div>
                </div>
              )}

              {/* Shared Honour Cards */}
              {dept.honour_cards && dept.honour_cards.length > 0 && (
                <div className="mt-4">
                  <p className="text-xs font-semibold text-gray-600 mb-2">
                    Shared Honour Programme Cards
                  </p>
                  <div className="space-y-2">
                    {dept.honour_cards.map((honourCard) => (
                      <div
                        key={honourCard.id}
                        className="bg-purple-50 rounded p-3 border border-purple-200"
                      >
                        <p className="text-sm font-semibold text-gray-900">
                          {honourCard.title}
                          {honourCard.semester_number && (
                            <span className="ml-2 text-xs text-gray-500">
                              (Semester {honourCard.semester_number})
                            </span>
                          )}
                        </p>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          );
        })}
      </div>
    );
  };

  if (loading) {
    return (
      <MainLayout title="Sharing Management" subtitle="Loading...">
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
            <p className="text-gray-600">Loading...</p>
          </div>
        </div>
      </MainLayout>
    );
  }

  return (
    <MainLayout
      title="Sharing Management"
      subtitle="Manage content sharing between cluster departments"
      actions={
        <button
          onClick={() => navigate("/dashboard")}
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
          <span>Back to Dashboard</span>
        </button>
      }
    >
      <div className="space-y-6">
        {/* Department Selector Modal */}
        {renderDepartmentSelector()}

        {/* Messages */}
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

        {success && (
          <div className="flex items-start space-x-3 p-4 bg-green-50 border border-green-200 rounded-lg">
            <svg
              className="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5"
              fill="currentColor"
              viewBox="0 0 20 20"
            >
              <path
                fillRule="evenodd"
                d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
                clipRule="evenodd"
              />
            </svg>
            <p className="text-sm font-medium text-green-600">{success}</p>
          </div>
        )}

        {/* Three Column Layout */}
        <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
          {/* Cluster Selection */}
          <div className="card-custom p-6">
            <h2 className="text-lg font-bold text-gray-900 mb-5">
              Select Cluster
            </h2>

            {clusters.length === 0 ? (
              <div className="text-center py-12 text-gray-500">
                <svg
                  className="w-12 h-12 text-gray-300 mx-auto mb-2"
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
                <p className="text-sm">No clusters available</p>
              </div>
            ) : (
              <div className="space-y-2">
                {clusters.map((cluster) => (
                  <button
                    key={cluster.id}
                    onClick={() => handleSelectCluster(cluster)}
                    className={`w-full text-left p-3 rounded-lg border-2 transition-all ${
                      selectedCluster?.id === cluster.id
                        ? "border-primary bg-primary-100"
                        : "bg-primary-200 hover:border-primary-300 hover:bg-primary-100"
                    }`}
                  >
                    <p className="font-semibold text-gray-900">
                      {cluster.name}
                    </p>
                    {cluster.description && (
                      <p className="text-xs text-gray-500 mt-1">
                        {cluster.description}
                      </p>
                    )}
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Department Selection */}
          <div className="card-custom p-6">
            <h2 className="text-lg font-bold text-gray-900 mb-5">
              {selectedCluster
                ? `Departments in ${selectedCluster.name}`
                : "Select a Cluster"}
            </h2>

            {!selectedCluster ? (
              <div className="text-center py-12 text-gray-500">
                <svg
                  className="w-12 h-12 text-gray-300 mx-auto mb-2"
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
                <p className="text-sm">Select a cluster first</p>
              </div>
            ) : clusterDepartments.length === 0 ? (
              <div className="text-center py-12 text-gray-500">
                <p className="text-sm">No departments in this cluster</p>
              </div>
            ) : (
              <div className="space-y-2">
                {clusterDepartments.map((dept) => (
                  <button
                    key={dept.department_id}
                    onClick={() => handleSelectCurriculum(dept)}
                    className={`w-full text-left p-3 rounded-lg border-2 transition-all ${
                      selectedCurriculum?.department_id === dept.department_id
                        ? "border-blue-500 bg-blue-50"
                        : "border-gray-200 hover:border-blue-300 hover:bg-gray-50"
                    }`}
                  >
                    <p className="font-semibold text-gray-900">
                      {dept.name || `Department ${dept.curriculum_id}`}
                    </p>
                    <p className="text-xs text-gray-500 mt-1">
                      ID: {dept.curriculum_id}
                    </p>
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Sharing Settings */}
          <div className="lg:col-span-2">
            {!selectedCluster ? (
              <div className="card-custom p-12 text-center">
                <svg
                  className="w-16 h-16 text-gray-300 mx-auto mb-3"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={1.5}
                    d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                  />
                </svg>
                <p className="text-sm text-gray-500">
                  Select a cluster to get started
                </p>
              </div>
            ) : !selectedCurriculum ? (
              <div className="card-custom p-12 text-center">
                <svg
                  className="w-16 h-16 text-gray-300 mx-auto mb-3"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={1.5}
                    d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                  />
                </svg>
                <p className="text-sm text-gray-500">
                  Select a curriculum to manage sharing
                </p>
              </div>
            ) : !sharingInfo ? (
              <div className="card-custom p-12 text-center">
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
                <p className="text-gray-600">Loading sharing information...</p>
              </div>
            ) : (
              <div className="space-y-6">
                {/* Content Sections */}
                <div className="card-custom p-6">
                  <h3 className="text-md font-bold text-gray-900 mb-4">
                    Mission Statements
                  </h3>
                  {renderItemList(sharingInfo.mission, "mission", "Mission")}
                </div>

                <div className="card-custom p-6">
                  <h3 className="text-md font-bold text-gray-900 mb-4">
                    Program Educational Objectives (PEOs)
                  </h3>
                  {renderItemList(sharingInfo.peos, "peos", "PEO")}
                </div>

                <div className="card-custom p-6">
                  <h3 className="text-md font-bold text-gray-900 mb-4">
                    Program Specific Outcomes (PSOs)
                  </h3>
                  {renderItemList(sharingInfo.psos, "psos", "PSO")}
                </div>

                {/* Normal Cards Section (Semesters, Verticals, Language Electives) */}
                {renderNormalCardsList()}

                {/* Honour Cards Section */}
                {renderHonourCardsList()}

                {/* Cluster Shared Content */}
                {renderClusterSharedContent()}
              </div>
            )}
          </div>
        </div>
      </div>
    </MainLayout>
  );
}

export default SharingManagementPage;
