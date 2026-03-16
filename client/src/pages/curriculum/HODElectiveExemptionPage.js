import React, { useCallback, useEffect, useState } from "react";
import MainLayout from "../../components/MainLayout";
import { API_BASE_URL } from "../../config";

const HODElectiveExemptionPage = () => {
  const userEmail = localStorage.getItem("userEmail");
  const [requests, setRequests] = useState([]);
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState("");
  const [processingId, setProcessingId] = useState(null);
  const [expandedRequestIds, setExpandedRequestIds] = useState([]);

  const toggleExpanded = (requestId) => {
    setExpandedRequestIds((prev) =>
      prev.includes(requestId)
        ? prev.filter((id) => id !== requestId)
        : [...prev, requestId],
    );
  };

  const renderValue = (value) => {
    if (value === null || value === undefined || value === "") return "-";
    return value;
  };

  const getFirstAvailableValue = (request, keys) => {
    for (const key of keys) {
      const value = request[key];
      if (value !== null && value !== undefined && value !== "") return value;
    }
    return "";
  };

  const fetchRequests = useCallback(async () => {
    if (!userEmail) return;

    setLoading(true);
    setMessage("");

    try {
      const response = await fetch(
        `${API_BASE_URL}/hod/elective-exemption-requests?email=${encodeURIComponent(userEmail)}`,
      );
      const data = await response.json();

      if (response.ok && data.success) {
        setRequests(data.requests || []);
        return;
      }

      setRequests([]);
      setMessage(data.message || "Failed to load exemption requests");
    } catch (error) {
      console.error("Error fetching elective exemption requests:", error);
      setRequests([]);
      setMessage("Failed to load exemption requests");
    } finally {
      setLoading(false);
    }
  }, [userEmail]);

  const handleStatusUpdate = async (requestId, status) => {
    setProcessingId(requestId);
    setMessage("");

    try {
      const response = await fetch(
        `${API_BASE_URL}/hod/elective-exemption-requests/${requestId}/status`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ email: userEmail, status }),
        },
      );

      const data = await response.json();
      if (!response.ok || !data.success) {
        throw new Error(data.message || `Failed to ${status} request`);
      }

      setMessage(
        status === "approved"
          ? `Request approved. Removed ${data.removed_choices || 0} matching student choice(s).`
          : "Request rejected.",
      );
      fetchRequests();
    } catch (error) {
      console.error(
        `Error updating exemption request status to ${status}:`,
        error,
      );
      setMessage(error.message || `Failed to ${status} request`);
    } finally {
      setProcessingId(null);
    }
  };

  useEffect(() => {
    fetchRequests();
  }, [fetchRequests]);

  return (
    <MainLayout
      title="Elective Excemption Requests"
      subtitle="Review and process student exemption requests for your department"
    >
      <div className="bg-white rounded-lg shadow-md p-4 md:p-6">
        <div className="flex items-center justify-between gap-4 flex-wrap mb-4">
          <div>
            <h3 className="text-lg font-bold text-gray-900">
              Student Elective Excemption Requests
            </h3>
            <p className="text-sm text-gray-500 mt-1">
              Approving a request will remove the corresponding PE choice from
              the student record.
            </p>
          </div>
          <button
            onClick={fetchRequests}
            disabled={loading}
            className="px-4 py-2 rounded-lg text-sm font-medium bg-primary text-white hover:bg-primary-600 disabled:opacity-60"
          >
            {loading ? "Refreshing..." : "Refresh Requests"}
          </button>
        </div>

        {message && (
          <div
            className={`mb-4 p-3 rounded-lg text-sm ${message.toLowerCase().includes("failed") ? "bg-red-50 text-red-700" : "bg-green-50 text-green-700"}`}
          >
            {message}
          </div>
        )}

        {loading ? (
          <div className="py-8 text-center text-sm text-gray-500">
            Loading exemption requests...
          </div>
        ) : requests.length === 0 ? (
          <div className="py-8 text-center text-sm text-gray-500">
            No exemption requests found for your department.
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[1000px] text-sm">
              <thead>
                <tr className="border-b bg-gray-50 text-left text-gray-600">
                  <th className="px-4 py-3 font-semibold">Student</th>
                  <th className="px-4 py-3 font-semibold">Request</th>
                  <th className="px-4 py-3 font-semibold">PE No</th>
                  <th className="px-4 py-3 font-semibold">Details</th>
                  <th className="px-4 py-3 font-semibold">Proof</th>
                  <th className="px-4 py-3 font-semibold">Status</th>
                  <th className="px-4 py-3 font-semibold">Action</th>
                </tr>
              </thead>
              <tbody>
                {requests.map((request) => {
                  const isExpanded = expandedRequestIds.includes(request.id);
                  const proofFileHref = request.certificate_file_path
                    ? `${API_BASE_URL.replace(/\/api$/, "")}${request.certificate_file_path}`
                    : "";

                  const industryAddress = getFirstAvailableValue(request, [
                    "industry_address",
                    "address",
                  ]);
                  const postalCode = getFirstAvailableValue(request, [
                    "postal_code",
                    "pincode",
                    "zip_code",
                    "zipcode",
                  ]);
                  const city = getFirstAvailableValue(request, ["city"]);
                  const state = getFirstAvailableValue(request, [
                    "state",
                    "province",
                  ]);
                  const country = getFirstAvailableValue(request, ["country"]);
                  const industryLink = getFirstAvailableValue(request, [
                    "industry_website_url",
                    "industry_link",
                    "industry_url",
                    "company_website",
                    "website",
                  ]);
                  const numberOfDaysAttended = getFirstAvailableValue(request, [
                    "number_of_days_attended",
                    "days_attended",
                    "duration_days",
                  ]);
                  const stipendAmount = getFirstAvailableValue(request, [
                    "stipend_amount",
                    "stipend",
                  ]);
                  const certificateLink = getFirstAvailableValue(request, [
                    "certificate_url",
                    "certificate_link",
                  ]);

                  const detailText =
                    request.request_type === "NPTEL"
                      ? `${request.online_course_name || "Online course"}${request.course_type ? ` - ${request.course_type}` : ""}`
                      : `${request.industry_name || "Industry"}${request.sector ? ` - ${request.sector}` : ""}${request.industry_contact ? ` - ${request.industry_contact}` : ""}`;

                  return (
                    <React.Fragment key={request.id}>
                      <tr className="border-b align-top hover:bg-gray-50 transition-colors">
                        <td className="px-4 py-3">
                          <button
                            type="button"
                            onClick={() => toggleExpanded(request.id)}
                            className="text-left w-full"
                          >
                            <div className="font-semibold text-gray-900 flex items-center gap-2">
                              <span>{isExpanded ? "▼" : "▶"}</span>
                              <span>{request.student_name}</span>
                            </div>
                            <div className="text-xs text-gray-500 mt-1 pl-6">
                              {request.enrollment_no ||
                                request.register_no ||
                                request.student_email}
                            </div>
                          </button>
                        </td>
                        <td className="px-4 py-3 text-gray-700">
                          <div className="font-medium">
                            {request.request_type}
                          </div>
                          <div className="text-xs text-gray-500 mt-1">
                            {request.created_at}
                          </div>
                        </td>
                        <td className="px-4 py-3">
                          <span className="inline-flex items-center px-2.5 py-1 rounded-full bg-background text-primary text-xs font-semibold">
                            PE {request.professional_elective_no}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-gray-700">
                          <div>{detailText}</div>
                          {(request.start_date || request.end_date) && (
                            <div className="text-xs text-gray-500 mt-1">
                              {request.start_date || "-"} to{" "}
                              {request.end_date || "-"}
                            </div>
                          )}
                        </td>
                        <td className="px-4 py-3">
                          <div className="flex flex-col gap-2">
                            {request.certificate_url && (
                              <a
                                href={request.certificate_url}
                                target="_blank"
                                rel="noreferrer"
                                className="text-primary hover:underline"
                              >
                                Certificate URL
                              </a>
                            )}
                            {proofFileHref && (
                              <a
                                href={proofFileHref}
                                target="_blank"
                                rel="noreferrer"
                                className="text-primary hover:underline"
                              >
                                Uploaded certificate
                              </a>
                            )}
                          </div>
                        </td>
                        <td className="px-4 py-3">
                          <span
                            className={`inline-flex items-center px-2.5 py-1 rounded-full text-xs font-semibold ${
                              request.status === "approved"
                                ? "bg-green-100 text-green-700"
                                : request.status === "rejected"
                                  ? "bg-red-100 text-red-700"
                                  : "bg-orange-100 text-orange-700"
                            }`}
                          >
                            {request.status}
                          </span>
                        </td>
                        <td className="px-4 py-3">
                          {request.status === "submitted" ? (
                            <div className="flex gap-2">
                              <button
                                onClick={() =>
                                  handleStatusUpdate(request.id, "approved")
                                }
                                disabled={processingId === request.id}
                                className="px-3 py-2 rounded-lg text-xs font-semibold bg-green-600 text-white hover:bg-green-700 disabled:opacity-60"
                              >
                                {processingId === request.id
                                  ? "Working..."
                                  : "Approve"}
                              </button>
                              <button
                                onClick={() =>
                                  handleStatusUpdate(request.id, "rejected")
                                }
                                disabled={processingId === request.id}
                                className="px-3 py-2 rounded-lg text-xs font-semibold bg-red-600 text-white hover:bg-red-700 disabled:opacity-60"
                              >
                                Reject
                              </button>
                            </div>
                          ) : (
                            <span className="text-xs text-gray-500">
                              Processed
                            </span>
                          )}
                        </td>
                      </tr>

                      {isExpanded && (
                        <tr className="border-b last:border-b-0 bg-gray-50">
                          <td colSpan={7} className="px-4 py-4">
                            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 text-sm">
                              <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500">
                                  Student Name
                                </p>
                                <p className="font-medium text-gray-900">
                                  {renderValue(request.student_name)}
                                </p>
                              </div>
                              <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500">
                                  Student Email
                                </p>
                                <p className="font-medium text-gray-900">
                                  {renderValue(request.student_email)}
                                </p>
                              </div>
                              <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500">
                                  Enrollment No
                                </p>
                                <p className="font-medium text-gray-900">
                                  {renderValue(request.enrollment_no)}
                                </p>
                              </div>
                              <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500">
                                  Register No
                                </p>
                                <p className="font-medium text-gray-900">
                                  {renderValue(request.register_no)}
                                </p>
                              </div>
                              <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500">
                                  Department
                                </p>
                                <p className="font-medium text-gray-900">
                                  {renderValue(request.department_name)}
                                </p>
                              </div>
                              <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500">
                                  Academic Year
                                </p>
                                <p className="font-medium text-gray-900">
                                  {renderValue(request.academic_year)}
                                </p>
                              </div>
                              <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500">
                                  Request Type
                                </p>
                                <p className="font-medium text-gray-900">
                                  {renderValue(request.request_type)}
                                </p>
                              </div>
                              <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500">
                                  Course Type
                                </p>
                                <p className="font-medium text-gray-900">
                                  {renderValue(request.course_type)}
                                </p>
                              </div>
                              <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500">
                                  Professional Elective
                                </p>
                                <p className="font-medium text-gray-900">
                                  PE{" "}
                                  {renderValue(
                                    request.professional_elective_no,
                                  )}
                                </p>
                              </div>
                              <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500">
                                  Online Course Name
                                </p>
                                <p className="font-medium text-gray-900">
                                  {renderValue(request.online_course_name)}
                                </p>
                              </div>
                              <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500">
                                  Industry Name
                                </p>
                                <p className="font-medium text-gray-900">
                                  {renderValue(request.industry_name)}
                                </p>
                              </div>
                              <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500">
                                  Industry Address
                                </p>
                                <p className="font-medium text-gray-900">
                                  {renderValue(industryAddress)}
                                </p>
                              </div>
                              <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500">
                                  City
                                </p>
                                <p className="font-medium text-gray-900">
                                  {renderValue(city)}
                                </p>
                              </div>
                              <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500">
                                  State
                                </p>
                                <p className="font-medium text-gray-900">
                                  {renderValue(state)}
                                </p>
                              </div>
                              <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500">
                                  Country
                                </p>
                                <p className="font-medium text-gray-900">
                                  {renderValue(country)}
                                </p>
                              </div>
                              <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500">
                                  Postal Code
                                </p>
                                <p className="font-medium text-gray-900">
                                  {renderValue(postalCode)}
                                </p>
                              </div>
                              <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500">
                                  Sector
                                </p>
                                <p className="font-medium text-gray-900">
                                  {renderValue(request.sector)}
                                </p>
                              </div>
                              <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500">
                                  Industry Contact
                                </p>
                                <p className="font-medium text-gray-900">
                                  {renderValue(request.industry_contact)}
                                </p>
                              </div>
                              <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500">
                                  Industry Link
                                </p>
                                {industryLink ? (
                                  <a
                                    href={industryLink}
                                    target="_blank"
                                    rel="noreferrer"
                                    className="font-medium text-primary hover:underline break-all"
                                  >
                                    {industryLink}
                                  </a>
                                ) : (
                                  <p className="font-medium text-gray-900">-</p>
                                )}
                              </div>
                              <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500">
                                  Start Date
                                </p>
                                <p className="font-medium text-gray-900">
                                  {renderValue(request.start_date)}
                                </p>
                              </div>
                              <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500">
                                  End Date
                                </p>
                                <p className="font-medium text-gray-900">
                                  {renderValue(request.end_date)}
                                </p>
                              </div>
                              <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500">
                                  Number of Days Attended
                                </p>
                                <p className="font-medium text-gray-900">
                                  {renderValue(numberOfDaysAttended)}
                                </p>
                              </div>
                              <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500">
                                  Stipend Amount
                                </p>
                                <p className="font-medium text-gray-900">
                                  {renderValue(stipendAmount)}
                                </p>
                              </div>
                              <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500">
                                  Certificate Link
                                </p>
                                {certificateLink ? (
                                  <a
                                    href={certificateLink}
                                    target="_blank"
                                    rel="noreferrer"
                                    className="font-medium text-primary hover:underline break-all"
                                  >
                                    {certificateLink}
                                  </a>
                                ) : (
                                  <p className="font-medium text-gray-900">-</p>
                                )}
                              </div>
                              <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500">
                                  Submitted At
                                </p>
                                <p className="font-medium text-gray-900">
                                  {renderValue(request.created_at)}
                                </p>
                              </div>
                            </div>
                          </td>
                        </tr>
                      )}
                    </React.Fragment>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </MainLayout>
  );
};

export default HODElectiveExemptionPage;
