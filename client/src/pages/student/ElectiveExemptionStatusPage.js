import React, { useCallback, useEffect, useState } from "react";
import MainLayout from "../../components/MainLayout";
import { API_BASE_URL } from "../../config";

function ElectiveExemptionStatusPage() {
  const [statusRequests, setStatusRequests] = useState([]);
  const [statusLoading, setStatusLoading] = useState(false);
  const [statusError, setStatusError] = useState("");

  const userEmail = localStorage.getItem("userEmail") || "";
  const userName = localStorage.getItem("userName") || "Student";

  const fetchRequestStatuses = useCallback(async () => {
    if (!userEmail) {
      setStatusRequests([]);
      return;
    }

    setStatusLoading(true);
    setStatusError("");
    try {
      const response = await fetch(
        `${API_BASE_URL}/students/elective-exemption-requests?email=${encodeURIComponent(userEmail)}`,
      );
      const data = await response.json();
      if (!response.ok || !data.success) {
        throw new Error(data.message || "Failed to load request statuses.");
      }
      setStatusRequests(data.requests || []);
    } catch (error) {
      setStatusRequests([]);
      setStatusError(error.message || "Failed to load request statuses.");
    } finally {
      setStatusLoading(false);
    }
  }, [userEmail]);

  useEffect(() => {
    fetchRequestStatuses();
  }, [fetchRequestStatuses]);

  return (
    <MainLayout
      title="Excemption Request Status"
      subtitle="Track your elective exemption request approval status"
    >
      <div className="flex w-full flex-col gap-6 py-8 px-6">
        <section className="card-custom rounded-3xl border border-primary-100 bg-gradient-to-r from-background via-white to-background p-6">
          <div className="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
            <div>
              <p className="text-sm font-semibold uppercase tracking-[0.24em] text-primary/70">
                Student status desk
              </p>
              <h2 className="mt-2 text-2xl font-semibold text-gray-900">
                {userName}, here is your exemption request history.
              </h2>
            </div>
            <div className="rounded-2xl border border-primary-100 bg-white px-4 py-3 text-sm text-gray-600 shadow-sm">
              Student email:{" "}
              <span className="font-medium text-primary">
                {userEmail || "Not available"}
              </span>
            </div>
          </div>
        </section>

        <section className="card-custom rounded-3xl border border-primary-100 bg-white p-6">
          <div className="mb-4 flex items-center justify-between gap-3">
            <div>
              <h3 className="text-xl font-semibold text-gray-900">
                My Excemption Request Status
              </h3>
              <p className="mt-1 text-sm text-gray-500">
                Track whether your requests are submitted, approved, or
                rejected.
              </p>
            </div>
            <button
              type="button"
              onClick={fetchRequestStatuses}
              className="btn-secondary-custom"
            >
              Refresh
            </button>
          </div>

          {statusError && (
            <div className="mb-4 rounded-2xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
              {statusError}
            </div>
          )}

          {statusLoading ? (
            <div className="rounded-2xl border border-primary-100 bg-background px-4 py-6 text-sm text-gray-600">
              Loading request statuses...
            </div>
          ) : statusRequests.length === 0 ? (
            <div className="rounded-2xl border border-primary-100 bg-background px-4 py-6 text-sm text-gray-600">
              No exemption requests submitted yet.
            </div>
          ) : (
            <div className="overflow-x-auto rounded-2xl border border-primary-100">
              <table className="w-full min-w-[900px] text-sm">
                <thead className="bg-background text-left text-gray-600">
                  <tr>
                    <th className="px-4 py-3 font-semibold">Request</th>
                    <th className="px-4 py-3 font-semibold">PE No</th>
                    <th className="px-4 py-3 font-semibold">Details</th>
                    <th className="px-4 py-3 font-semibold">Submitted</th>
                    <th className="px-4 py-3 font-semibold">Status</th>
                  </tr>
                </thead>
                <tbody>
                  {statusRequests.map((request) => {
                    const detailText =
                      request.request_type === "NPTEL"
                        ? `${request.online_course_name || "Online course"}${request.course_type ? ` - ${request.course_type}` : ""}`
                        : `${request.industry_name || "Industry"}${request.sector ? ` - ${request.sector}` : ""}${request.industry_contact ? ` - ${request.industry_contact}` : ""}`;

                    return (
                      <tr
                        key={request.id}
                        className="border-t border-primary-100"
                      >
                        <td className="px-4 py-3 font-medium text-gray-900">
                          {request.request_type}
                        </td>
                        <td className="px-4 py-3 text-gray-700">
                          PE {request.professional_elective_no}
                        </td>
                        <td className="px-4 py-3 text-gray-700">
                          <div>{detailText}</div>
                          {(request.start_date || request.end_date) && (
                            <div className="mt-1 text-xs text-gray-500">
                              {request.start_date || "-"} to{" "}
                              {request.end_date || "-"}
                            </div>
                          )}
                        </td>
                        <td className="px-4 py-3 text-gray-600">
                          {request.created_at || "-"}
                        </td>
                        <td className="px-4 py-3">
                          <span
                            className={`inline-flex items-center rounded-full px-2.5 py-1 text-xs font-semibold ${
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
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </section>
      </div>
    </MainLayout>
  );
}

export default ElectiveExemptionStatusPage;
