import React, { useMemo, useState } from "react";
import MainLayout from "../../components/MainLayout";
import { API_BASE_URL } from "../../config";

const inputClassName =
  "input-custom text-sm text-gray-900 placeholder:text-gray-400 border border-primary-100 focus:border-primary focus:ring-2 focus:ring-primary/20";

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
    <>
      <div className="grid gap-4 md:grid-cols-2">
        <label className="flex flex-col gap-2 text-sm font-medium text-gray-700">
          Certificate URL
          <input
            type="url"
            name="certificate_url"
            value={formData.certificate_url}
            onChange={handleChange}
            placeholder="https://..."
            className={inputClassName}
          />
        </label>
      </div>

      <label className="flex flex-col gap-2 text-sm font-medium text-gray-700">
        Certificate proof upload
        <input
          key={fileInputKey}
          type="file"
          accept=".pdf,.png,.jpg,.jpeg"
          onChange={handleFileChange}
          className="w-full rounded-xl border border-dashed border-primary-200 bg-background px-3 py-3 text-sm text-gray-700 file:mr-4 file:rounded-lg file:border-0 file:bg-primary file:px-3 file:py-2 file:text-sm file:font-medium file:text-white hover:file:bg-primary-600"
        />
      </label>
    </>
  );

  const renderNptelFields = () => (
    <div className="grid gap-4 md:grid-cols-2">
      <label className="flex flex-col gap-2 text-sm font-medium text-gray-700 md:col-span-2">
        Online course name
        <input
          type="text"
          name="online_course_name"
          value={formData.online_course_name}
          onChange={handleChange}
          className={inputClassName}
        />
      </label>

      <label className="flex flex-col gap-2 text-sm font-medium text-gray-700">
        Course type
        <input
          type="text"
          name="course_type"
          value={formData.course_type}
          onChange={handleChange}
          placeholder="Swayam-NPTEL"
          className={inputClassName}
        />
      </label>

      <label className="flex flex-col gap-2 text-sm font-medium text-gray-700">
        Professional elective number
        <select
          name="professional_elective_no"
          value={formData.professional_elective_no}
          onChange={handleChange}
          className={inputClassName}
        >
          {professionalElectiveOptions.map((option) => (
            <option key={option} value={option}>
              Professional Elective {option}
            </option>
          ))}
        </select>
      </label>

      <label className="flex flex-col gap-2 text-sm font-medium text-gray-700">
        Course duration in weeks
        <input
          type="number"
          min="1"
          name="course_duration_weeks"
          value={formData.course_duration_weeks}
          onChange={handleChange}
          className={inputClassName}
        />
      </label>

      <label className="flex flex-col gap-2 text-sm font-medium text-gray-700">
        Start date
        <input
          type="date"
          name="start_date"
          value={formData.start_date}
          onChange={handleChange}
          className={inputClassName}
        />
      </label>

      <label className="flex flex-col gap-2 text-sm font-medium text-gray-700">
        End date
        <input
          type="date"
          name="end_date"
          value={formData.end_date}
          onChange={handleChange}
          className={inputClassName}
        />
      </label>
    </div>
  );

  const renderInternshipFields = () => (
    <div className="grid gap-4 md:grid-cols-2">
      <label className="flex flex-col gap-2 text-sm font-medium text-gray-700">
        Industry name
        <input
          type="text"
          name="industry_name"
          value={formData.industry_name}
          onChange={handleChange}
          className={inputClassName}
        />
      </label>

      <label className="flex flex-col gap-2 text-sm font-medium text-gray-700">
        Professional elective number
        <select
          name="professional_elective_no"
          value={formData.professional_elective_no}
          onChange={handleChange}
          className={inputClassName}
        >
          {professionalElectiveOptions.map((option) => (
            <option key={option} value={option}>
              Professional Elective {option}
            </option>
          ))}
        </select>
      </label>

      <label className="flex flex-col gap-2 text-sm font-medium text-gray-700">
        Industry contact
        <input
          type="text"
          name="industry_contact"
          value={formData.industry_contact}
          onChange={handleChange}
          placeholder="Contact person / number"
          className={inputClassName}
        />
      </label>

      <label className="flex flex-col gap-2 text-sm font-medium text-gray-700">
        Sector
        <select
          name="sector"
          value={formData.sector}
          onChange={handleChange}
          className={inputClassName}
        >
          <option value="">Select sector</option>
          {internshipSectorOptions.map((option) => (
            <option key={option} value={option}>
              {option}
            </option>
          ))}
        </select>
      </label>

      <label className="flex flex-col gap-2 text-sm font-medium text-gray-700 md:col-span-2">
        Industry address
        <textarea
          name="industry_address"
          value={formData.industry_address}
          onChange={handleChange}
          rows="3"
          className={`${inputClassName} resize-y min-h-[96px]`}
        />
      </label>

      <label className="flex flex-col gap-2 text-sm font-medium text-gray-700">
        City
        <input
          type="text"
          name="city"
          value={formData.city}
          onChange={handleChange}
          className={inputClassName}
        />
      </label>

      <label className="flex flex-col gap-2 text-sm font-medium text-gray-700">
        State
        <input
          type="text"
          name="state"
          value={formData.state}
          onChange={handleChange}
          className={inputClassName}
        />
      </label>

      <label className="flex flex-col gap-2 text-sm font-medium text-gray-700">
        Postal code
        <input
          type="text"
          name="postal_code"
          value={formData.postal_code}
          onChange={handleChange}
          className={inputClassName}
        />
      </label>

      <label className="flex flex-col gap-2 text-sm font-medium text-gray-700">
        Country
        <input
          type="text"
          name="country"
          value={formData.country}
          onChange={handleChange}
          className={inputClassName}
        />
      </label>

      <label className="flex flex-col gap-2 text-sm font-medium text-gray-700 md:col-span-2">
        Industry website link
        <input
          type="url"
          name="industry_website_url"
          value={formData.industry_website_url}
          onChange={handleChange}
          placeholder="https://company.example"
          className={inputClassName}
        />
      </label>

      <label className="flex flex-col gap-2 text-sm font-medium text-gray-700">
        Start date
        <input
          type="date"
          name="start_date"
          value={formData.start_date}
          onChange={handleChange}
          className={inputClassName}
        />
      </label>

      <label className="flex flex-col gap-2 text-sm font-medium text-gray-700">
        End date
        <input
          type="date"
          name="end_date"
          value={formData.end_date}
          onChange={handleChange}
          className={inputClassName}
        />
      </label>

      <label className="flex flex-col gap-2 text-sm font-medium text-gray-700">
        Number of days attended
        <input
          type="number"
          min="1"
          name="number_of_days_attended"
          value={formData.number_of_days_attended}
          onChange={handleChange}
          className={inputClassName}
        />
      </label>

      <label className="flex flex-col gap-2 text-sm font-medium text-gray-700">
        Stipend amount
        <input
          type="number"
          min="0"
          step="0.01"
          name="stipend_amount"
          value={formData.stipend_amount}
          onChange={handleChange}
          className={inputClassName}
        />
      </label>
    </div>
  );

  return (
    <MainLayout
      title="Elective Excemption"
      subtitle="Submit exemption requests for elective slots in semesters 4 to 8."
    >
      <div className="flex w-full flex-col gap-6 py-8 px-6">
        <section className="card-custom rounded-3xl border border-primary-100 bg-gradient-to-r from-background via-white to-background p-6">
          <div className="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
            <div>
              <p className="text-sm font-semibold uppercase tracking-[0.24em] text-primary/70">
                Student request desk
              </p>
              <h2 className="mt-2 text-2xl font-semibold text-gray-900">
                {userName}, choose the exemption request you want to submit.
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

        <section className="grid gap-4 md:grid-cols-3">
          {requestTypeOptions.map((option) => {
            const isSelected = option.key === selectedType;
            return (
              <button
                key={option.key}
                type="button"
                disabled={!option.enabled}
                onClick={() => option.enabled && setSelectedType(option.key)}
                className={`card-custom rounded-3xl border p-5 text-left transition-all duration-200 ${
                  isSelected
                    ? "border-primary bg-primary text-white shadow-lg"
                    : "border-primary-100 bg-white text-gray-900 hover:border-primary"
                } ${option.enabled ? "hover:-translate-y-0.5" : "cursor-not-allowed opacity-60"}`}
              >
                <div className="flex items-center justify-between gap-3">
                  <h3 className="text-lg font-semibold">{option.label}</h3>
                  {!option.enabled && (
                    <span className="rounded-full bg-background px-3 py-1 text-xs font-semibold uppercase tracking-[0.18em] text-primary/70">
                      Soon
                    </span>
                  )}
                </div>
                <p
                  className={`mt-3 text-sm ${isSelected ? "text-primary-100" : "text-gray-600"}`}
                >
                  {option.description}
                </p>
              </button>
            );
          })}
        </section>

        {message.text && (
          <div
            className={`rounded-2xl border px-4 py-3 text-sm ${
              message.type === "success"
                ? "border-primary-200 bg-background text-primary"
                : "border-red-200 bg-red-50 text-red-700"
            }`}
          >
            {message.text}
          </div>
        )}

        <form
          onSubmit={handleSubmit}
          className="card-custom rounded-3xl border border-primary-100 bg-white p-6"
        >
          <div className="mb-6 flex items-center justify-between gap-3">
            <div>
              <h3 className="text-xl font-semibold text-gray-900">
                {activeOption?.label}
              </h3>
              <p className="mt-1 text-sm text-gray-500">
                Fill the fields carefully. Attach the certificate file,
                certificate URL, or both.
              </p>
            </div>
            <span className="rounded-full bg-background px-3 py-1 text-xs font-semibold uppercase tracking-[0.18em] text-primary border border-primary-100">
              Semesters 4-8
            </span>
          </div>

          <div className="space-y-6">
            {selectedType === "NPTEL" && renderNptelFields()}
            {selectedType === "INTERNSHIP" && renderInternshipFields()}
            {renderCommonFields()}
          </div>

          <div className="mt-8 flex flex-col gap-3 border-t border-primary-100 pt-6 md:flex-row md:items-center md:justify-between">
            <p className="text-sm text-gray-500">
              Both forms use professional elective number 1 to 9.
            </p>
            <div className="flex gap-3">
              <button
                type="button"
                onClick={resetForm}
                className="btn-secondary-custom"
              >
                Reset
              </button>
              <button
                type="submit"
                disabled={submitting || !activeOption?.enabled}
                className="btn-primary-custom disabled:cursor-not-allowed disabled:opacity-60"
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
