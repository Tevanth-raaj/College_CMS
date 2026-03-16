import React, { useMemo, useState } from "react";
import MainLayout from "../../components/MainLayout";
import { API_BASE_URL } from "../../config";

const adminInputClassName =
  "w-full rounded-lg border border-primary-100 bg-white px-4 py-3 text-sm text-gray-900 placeholder:text-gray-400 shadow-sm transition focus:border-primary focus:outline-none focus:ring-4 focus:ring-primary/10";

const adminPrimaryBtnClass =
  "inline-flex items-center justify-center rounded-lg bg-primary px-5 py-3 text-sm font-semibold text-white shadow-sm transition hover:bg-primary-600 disabled:cursor-not-allowed disabled:opacity-60";

const adminSecondaryBtnClass =
  "inline-flex items-center justify-center rounded-lg border border-primary-100 bg-white px-5 py-3 text-sm font-semibold text-primary shadow-sm transition hover:border-primary hover:bg-primary-50";

const adminSectionCardClass =
  "card-custom rounded-lg border border-primary-100 bg-white p-6 shadow-sm";

const labelClassName =
  "mb-2 block text-xs font-semibold uppercase tracking-[0.18em] text-slate-500";

const requestTypeOptions = [
  {
    key: "NPTEL",
    label: "Online Course [NPTEL]",
    description: "Submit an exemption request for a certified online course.",
    enabled: true,
  },
  {
    key: "INTERNSHIP",
    label: "Internship / Industrial Training",
    description:
      "Submit internship or industrial training details for exemption.",
    enabled: true,
  },
  {
    key: "THREE_ONE_CREDIT",
    label: "Three One Credit Courses",
    description: "Kept aside for now until the required fields are finalized.",
    enabled: false,
  },
];

const professionalElectiveOptions = [1, 2, 3, 4, 5, 6, 7, 8, 9];
const internshipSectorOptions = ["Government", "Private"];

const initialFormState = {
  professional_elective_no: "1",
  online_course_name: "",
  course_type: "",
  start_date: "",
  end_date: "",
  course_duration_weeks: "",
  certificate_url: "",
  industry_name: "",
  industry_contact: "",
  sector: "",
  industry_address: "",
  city: "",
  state: "",
  postal_code: "",
  country: "",
  industry_website_url: "",
  number_of_days_attended: "",
  stipend_amount: "",
};

function ElectiveExemptionPage() {
  const [selectedType, setSelectedType] = useState("NPTEL");
  const [formData, setFormData] = useState(initialFormState);
  const [certificateFile, setCertificateFile] = useState(null);
  const [fileInputKey, setFileInputKey] = useState(0);
  const [submitting, setSubmitting] = useState(false);
  const [message, setMessage] = useState({ type: "", text: "" });

  const userEmail = localStorage.getItem("userEmail") || "";
  const userName = localStorage.getItem("userName") || "Student";

  const activeOption = useMemo(
    () => requestTypeOptions.find((option) => option.key === selectedType),
    [selectedType],
  );

  const handleChange = (event) => {
    const { name, value } = event.target;
    setFormData((current) => ({ ...current, [name]: value }));
  };

  const handleFileChange = (event) => {
    const file = event.target.files?.[0] || null;
    setCertificateFile(file);
  };

  const resetForm = () => {
    setFormData(initialFormState);
    setCertificateFile(null);
    setFileInputKey((current) => current + 1);
  };

  const validateCommonFields = () => {
    if (!userEmail) {
      return "Student email is not available in the current session.";
    }

    if (!formData.certificate_url.trim() && !certificateFile) {
      return "Attach a certificate file or provide the certificate URL.";
    }

    if (formData.start_date && formData.end_date) {
      const startDate = new Date(formData.start_date);
      const endDate = new Date(formData.end_date);
      if (endDate < startDate) {
        return "End date must be on or after the start date.";
      }
    }

    return "";
  };

  const validateCurrentForm = () => {
    const commonError = validateCommonFields();
    if (commonError) {
      return commonError;
    }

    if (selectedType === "NPTEL") {
      if (
        !formData.professional_elective_no ||
        !formData.online_course_name.trim() ||
        !formData.course_type.trim() ||
        !formData.start_date ||
        !formData.end_date ||
        !formData.course_duration_weeks
      ) {
        return "Fill all required NPTEL fields before submitting.";
      }

      if (Number(formData.course_duration_weeks) <= 0) {
        return "Course duration in weeks must be greater than zero.";
      }

      if (
        Number(formData.professional_elective_no) < 1 ||
        Number(formData.professional_elective_no) > 9
      ) {
        return "Professional elective number must be between 1 and 9.";
      }
    }

    if (selectedType === "INTERNSHIP") {
      if (!formData.professional_elective_no) {
        return "Select the professional elective number.";
      }

      const requiredFields = [
        formData.industry_name,
        formData.industry_contact,
        formData.sector,
        formData.industry_address,
        formData.city,
        formData.state,
        formData.postal_code,
        formData.country,
        formData.industry_website_url,
        formData.start_date,
        formData.end_date,
        formData.number_of_days_attended,
        formData.stipend_amount,
      ];

      if (requiredFields.some((value) => `${value}`.trim() === "")) {
        return "Fill all required internship fields before submitting.";
      }

      if (Number(formData.number_of_days_attended) <= 0) {
        return "Number of days attended must be greater than zero.";
      }

      if (Number(formData.stipend_amount) < 0) {
        return "Stipend amount cannot be negative.";
      }

      if (
        Number(formData.professional_elective_no) < 1 ||
        Number(formData.professional_elective_no) > 9
      ) {
        return "Professional elective number must be between 1 and 9.";
      }
    }

    return "";
  };

  const handleSubmit = async (event) => {
    event.preventDefault();

    if (!activeOption?.enabled) {
      setMessage({
        type: "error",
        text: "This request form is not open yet.",
      });
      return;
    }

    const validationError = validateCurrentForm();
    if (validationError) {
      setMessage({ type: "error", text: validationError });
      return;
    }

    setSubmitting(true);
    setMessage({ type: "", text: "" });

    try {
      const payload = new FormData();
      payload.append("student_email", userEmail);
      payload.append("request_type", selectedType);
      payload.append("certificate_url", formData.certificate_url.trim());

      if (certificateFile) {
        payload.append("certificate_file", certificateFile);
      }

      if (selectedType === "NPTEL") {
        payload.append(
          "professional_elective_no",
          formData.professional_elective_no,
        );
        payload.append(
          "online_course_name",
          formData.online_course_name.trim(),
        );
        payload.append("course_type", formData.course_type.trim());
        payload.append("start_date", formData.start_date);
        payload.append("end_date", formData.end_date);
        payload.append("course_duration_weeks", formData.course_duration_weeks);
      }

      if (selectedType === "INTERNSHIP") {
        payload.append(
          "professional_elective_no",
          formData.professional_elective_no,
        );
        payload.append("industry_name", formData.industry_name.trim());
        payload.append("industry_contact", formData.industry_contact.trim());
        payload.append("sector", formData.sector.trim());
        payload.append("industry_address", formData.industry_address.trim());
        payload.append("city", formData.city.trim());
        payload.append("state", formData.state.trim());
        payload.append("postal_code", formData.postal_code.trim());
        payload.append("country", formData.country.trim());
        payload.append(
          "industry_website_url",
          formData.industry_website_url.trim(),
        );
        payload.append("start_date", formData.start_date);
        payload.append("end_date", formData.end_date);
        payload.append(
          "number_of_days_attended",
          formData.number_of_days_attended,
        );
        payload.append("stipend_amount", formData.stipend_amount);
      }

      const response = await fetch(
        `${API_BASE_URL}/students/elective-exemption-requests`,
        {
          method: "POST",
          body: payload,
        },
      );

      const responseText = await response.text();
      if (!response.ok) {
        throw new Error(responseText || "Failed to submit request");
      }

      setMessage({
        type: "success",
        text: `Request submitted successfully for ${activeOption.label}.`,
      });
      resetForm();
    } catch (error) {
      setMessage({
        type: "error",
        text: error.message || "Failed to submit request.",
      });
    } finally {
      setSubmitting(false);
    }
  };

  const renderCommonFields = () => (
    <div className="grid gap-5 lg:grid-cols-2">
      <div>
        <label className={labelClassName}>Professional elective number</label>
        <select
          name="professional_elective_no"
          value={formData.professional_elective_no}
          onChange={handleChange}
          className={adminInputClassName}
        >
          {professionalElectiveOptions.map((electiveNo) => (
            <option key={electiveNo} value={electiveNo}>
              Professional Elective {electiveNo}
            </option>
          ))}
        </select>
      </div>

      <div>
        <label className={labelClassName}>Certificate URL</label>
        <input
          type="url"
          name="certificate_url"
          value={formData.certificate_url}
          onChange={handleChange}
          placeholder="https://..."
          className={adminInputClassName}
        />
      </div>

      <div className="lg:col-span-2">
        <label className={labelClassName}>Certificate proof upload</label>
        <input
          key={fileInputKey}
          id="elective-exemption-certificate"
          type="file"
          accept=".pdf,.png,.jpg,.jpeg"
          onChange={handleFileChange}
          className="hidden"
        />
        <div className="flex min-w-0 items-center gap-4 rounded-lg border border-dashed border-primary-200 bg-gradient-to-r from-primary-50 to-white px-4 py-4">
          <label
            htmlFor="elective-exemption-certificate"
            className={`${adminPrimaryBtnClass} cursor-pointer whitespace-nowrap`}
          >
            Choose File
          </label>
          <div className="min-w-0 flex-1">
            <p className="truncate text-sm font-medium text-gray-700">
              {certificateFile ? certificateFile.name : "No file chosen"}
            </p>
            <p className="mt-0.5 text-xs text-gray-500">
              PDF, PNG, JPG, or JPEG up to 10 MB.
            </p>
          </div>
        </div>
      </div>
    </div>
  );

  const renderNptelFields = () => (
    <div className={`${adminSectionCardClass} grid gap-5 md:grid-cols-2`}>
      <div className="md:col-span-2">
        <h4 className="text-base font-semibold text-gray-900">
          Online course details
        </h4>
        <p className="mt-1 text-sm text-gray-500">
          Enter the certified course information exactly as it appears on the
          completion proof.
        </p>
      </div>
      <div className="md:col-span-2">
        <label className={labelClassName}>Online course name</label>
        <input
          type="text"
          name="online_course_name"
          value={formData.online_course_name}
          onChange={handleChange}
          className={adminInputClassName}
        />
      </div>

      <div>
        <label className={labelClassName}>Course type</label>
        <input
          type="text"
          name="course_type"
          value={formData.course_type}
          onChange={handleChange}
          placeholder="Elite / Domain / Certification"
          className={adminInputClassName}
        />
      </div>

      <div>
        <label className={labelClassName}>Course duration in weeks</label>
        <input
          type="number"
          min="1"
          name="course_duration_weeks"
          value={formData.course_duration_weeks}
          onChange={handleChange}
          className={adminInputClassName}
        />
      </div>

      <div>
        <label className={labelClassName}>Start date</label>
        <input
          type="date"
          name="start_date"
          value={formData.start_date}
          onChange={handleChange}
          className={adminInputClassName}
        />
      </div>

      <div>
        <label className={labelClassName}>End date</label>
        <input
          type="date"
          name="end_date"
          value={formData.end_date}
          onChange={handleChange}
          className={adminInputClassName}
        />
      </div>
    </div>
  );

  const renderInternshipFields = () => (
    <div className={`${adminSectionCardClass} grid gap-5 md:grid-cols-2`}>
      <div className="md:col-span-2">
        <h4 className="text-base font-semibold text-gray-900">
          Internship details
        </h4>
        <p className="mt-1 text-sm text-gray-500">
          Provide the organization, duration, and attendance details for the
          internship or industrial training.
        </p>
      </div>
      <div>
        <label className={labelClassName}>Industry name</label>
        <input
          type="text"
          name="industry_name"
          value={formData.industry_name}
          onChange={handleChange}
          className={adminInputClassName}
        />
      </div>

      <div>
        <label className={labelClassName}>Industry contact</label>
        <input
          type="text"
          name="industry_contact"
          value={formData.industry_contact}
          onChange={handleChange}
          placeholder="Contact person / number"
          className={adminInputClassName}
        />
      </div>

      <div>
        <label className={labelClassName}>Sector</label>
        <select
          name="sector"
          value={formData.sector}
          onChange={handleChange}
          className={adminInputClassName}
        >
          <option value="">Select sector</option>
          {internshipSectorOptions.map((sectorOption) => (
            <option key={sectorOption} value={sectorOption}>
              {sectorOption}
            </option>
          ))}
        </select>
      </div>

      <div className="md:col-span-2">
        <label className={labelClassName}>Industry address</label>
        <textarea
          name="industry_address"
          value={formData.industry_address}
          onChange={handleChange}
          rows="3"
          className={`${adminInputClassName} min-h-[96px] resize-y`}
        />
      </div>

      <div>
        <label className={labelClassName}>City</label>
        <input
          type="text"
          name="city"
          value={formData.city}
          onChange={handleChange}
          className={adminInputClassName}
        />
      </div>

      <div>
        <label className={labelClassName}>State</label>
        <input
          type="text"
          name="state"
          value={formData.state}
          onChange={handleChange}
          className={adminInputClassName}
        />
      </div>

      <div>
        <label className={labelClassName}>Postal code</label>
        <input
          type="text"
          name="postal_code"
          value={formData.postal_code}
          onChange={handleChange}
          className={adminInputClassName}
        />
      </div>

      <div>
        <label className={labelClassName}>Country</label>
        <input
          type="text"
          name="country"
          value={formData.country}
          onChange={handleChange}
          className={adminInputClassName}
        />
      </div>

      <div className="md:col-span-2">
        <label className={labelClassName}>Industry website link</label>
        <input
          type="url"
          name="industry_website_url"
          value={formData.industry_website_url}
          onChange={handleChange}
          placeholder="https://company.example"
          className={adminInputClassName}
        />
      </div>

      <div>
        <label className={labelClassName}>Start date</label>
        <input
          type="date"
          name="start_date"
          value={formData.start_date}
          onChange={handleChange}
          className={adminInputClassName}
        />
      </div>

      <div>
        <label className={labelClassName}>End date</label>
        <input
          type="date"
          name="end_date"
          value={formData.end_date}
          onChange={handleChange}
          className={adminInputClassName}
        />
      </div>

      <div>
        <label className={labelClassName}>Number of days attended</label>
        <input
          type="number"
          min="1"
          name="number_of_days_attended"
          value={formData.number_of_days_attended}
          onChange={handleChange}
          className={adminInputClassName}
        />
      </div>

      <div>
        <label className={labelClassName}>Stipend amount</label>
        <input
          type="number"
          min="0"
          step="0.01"
          name="stipend_amount"
          value={formData.stipend_amount}
          onChange={handleChange}
          className={adminInputClassName}
        />
      </div>
    </div>
  );

  return (
    <MainLayout
      title="Elective Exemption"
      subtitle="Submit exemption requests for elective slots in semesters 4 to 8."
    >
      <div className="flex w-full flex-col gap-6 px-6 py-8">
        <div className={adminSectionCardClass}>
          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div>
              <p className="text-sm font-semibold uppercase tracking-[0.2em] text-primary/70">
                Student request desk
              </p>
              <h2 className="mt-2 text-2xl font-semibold text-gray-900">
                {userName}, submit your elective exemption request.
              </h2>
            </div>
            <div className="rounded-lg border border-primary-100 bg-primary-50 px-4 py-3 text-sm text-gray-600">
              Student email:{" "}
              <span className="font-medium text-primary">
                {userEmail || "Not available"}
              </span>
            </div>
          </div>
        </div>

        <div className={adminSectionCardClass}>
          <div className="mb-4 flex items-center justify-between gap-3">
            <div>
              <h3 className="text-lg font-semibold text-gray-900">
                Request type
              </h3>
              <p className="mt-1 text-sm text-gray-500">
                Choose the exemption category you want to apply for.
              </p>
            </div>
          </div>
          <div className="grid gap-4 md:grid-cols-3">
            {requestTypeOptions.map((option) => {
              const isSelected = option.key === selectedType;
              return (
                <button
                  key={option.key}
                  type="button"
                  disabled={!option.enabled}
                  onClick={() => option.enabled && setSelectedType(option.key)}
                  className={`rounded-lg border p-5 text-left transition-all ${
                    isSelected
                      ? "border-primary bg-primary-50 text-primary shadow-sm"
                      : "border-primary-100 bg-white text-gray-900 hover:border-primary hover:bg-primary-50/40"
                  } ${option.enabled ? "" : "cursor-not-allowed opacity-60"}`}
                >
                  <div className="flex items-center justify-between gap-3">
                    <h4 className="text-base font-semibold">{option.label}</h4>
                    {!option.enabled && (
                      <span className="rounded-md bg-gray-100 px-2.5 py-1 text-[11px] font-semibold uppercase tracking-[0.14em] text-gray-500">
                        Soon
                      </span>
                    )}
                  </div>
                  <p className="mt-3 text-sm text-gray-600">
                    {option.description}
                  </p>
                </button>
              );
            })}
          </div>
        </div>

        {message.text && (
          <div
            className={`rounded-lg border px-4 py-3 text-sm ${
              message.type === "success"
                ? "border-primary-200 bg-primary-50 text-primary"
                : "border-red-200 bg-red-50 text-red-700"
            }`}
          >
            {message.text}
          </div>
        )}

        <form
          onSubmit={handleSubmit}
          className={`${adminSectionCardClass} space-y-6`}
        >
          <div className="border-b border-primary-100 pb-5">
            <h3 className="text-xl font-semibold text-gray-900">
              {activeOption?.label}
            </h3>
            <p className="mt-1 text-sm text-gray-500">
              Fill the required details and attach the supporting proof.
            </p>
          </div>

          {selectedType === "NPTEL" && renderNptelFields()}
          {selectedType === "INTERNSHIP" && renderInternshipFields()}

          <div className={adminSectionCardClass}>
            <div className="mb-5">
              <h4 className="text-base font-semibold text-gray-900">
                Proof and elective mapping
              </h4>
              <p className="mt-1 text-sm text-gray-500">
                Select the semester slot and provide the certificate proof for
                review.
              </p>
            </div>
            {renderCommonFields()}
          </div>

          <div className="flex flex-col gap-3 border-t border-primary-100 pt-6 md:flex-row md:items-center md:justify-between">
            <p className="text-sm text-gray-500">
              Applicable only for elective slots in semesters 4 to 8.
            </p>
            <div className="flex flex-wrap gap-3">
              <button
                type="button"
                onClick={resetForm}
                className={adminSecondaryBtnClass}
              >
                Reset
              </button>
              <button
                type="submit"
                disabled={submitting || !activeOption?.enabled}
                className={adminPrimaryBtnClass}
              >
                {submitting ? "Submitting..." : "Submit Request"}
              </button>
            </div>
          </div>
        </form>
      </div>
    </MainLayout>
  );
}

export default ElectiveExemptionPage;
