#!/usr/bin/env python3
import argparse
import csv
import json
import os
import sys
from collections import defaultdict
from typing import Any, Dict, List, Optional, Tuple
from urllib import error, request


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description=(
            "Auto-insert courses into Semester 4 using Management Portal API endpoints. "
            "Supports dry-run mode and duplicate-safe inserts."
        )
    )

    parser.add_argument(
        "--base-url",
        required=True,
        help="API base URL, e.g. http://localhost:8080",
    )
    parser.add_argument(
        "--auth-token",
        default="",
        help="Optional bearer token for protected endpoints.",
    )

    parser.add_argument(
        "--courses-file",
        required=True,
        help="Path to JSON/CSV describing courses to insert."
        " Each row should include curriculum_id and course_id for best results.",
    )
    parser.add_argument(
        "--curriculum-id",
        type=int,
        default=0,
        help="Default curriculum_id for rows that do not define curriculum_id.",
    )
    parser.add_argument(
        "--semester-number",
        type=int,
        default=4,
        help="Semester number to insert into (default: 4).",
    )
    parser.add_argument(
        "--resolve-existing-by-semester",
        type=int,
        default=4,
        help=(
            "When course_id is missing, resolve it by course_code using "
            "/api/all-departments/semester/{n}/courses (default: 4)."
        ),
    )
    parser.add_argument(
        "--default-course-type",
        default="1",
        help=(
            "Fallback course_type used when creating a course and course_type is absent in input. "
            "Can be numeric ID or name. Default: 1"
        ),
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Show what would be inserted, without calling write endpoints.",
    )
    return parser.parse_args()


def load_courses(file_path: str) -> List[Dict[str, Any]]:
    if not os.path.exists(file_path):
        raise FileNotFoundError(f"courses-file not found: {file_path}")

    ext = os.path.splitext(file_path)[1].lower()
    if ext == ".json":
        with open(file_path, "r", encoding="utf-8") as f:
            data = json.load(f)
        if not isinstance(data, list):
            raise ValueError("JSON courses file must be an array of objects.")
        return [normalize_course_row(row, idx + 1) for idx, row in enumerate(data)]

    if ext == ".csv":
        with open(file_path, "r", encoding="utf-8-sig") as f:
            preview = f.readline().strip()
        if "Code No." in preview and "Hours/Week" in preview:
            return load_course_template_csv(file_path)

        rows: List[Dict[str, Any]] = []
        with open(file_path, "r", encoding="utf-8-sig", newline="") as f:
            reader = csv.DictReader(f)
            for idx, row in enumerate(reader, start=1):
                rows.append(normalize_course_row(row, idx))
        return rows

    raise ValueError("courses-file must be .json or .csv")


def load_course_template_csv(file_path: str) -> List[Dict[str, Any]]:
    rows: List[Dict[str, Any]] = []
    with open(file_path, "r", encoding="utf-8-sig") as f:
        lines = [line.rstrip("\n") for line in f if line.strip()]

    if not lines:
        return rows

    for idx, line in enumerate(lines[1:], start=1):
        parts = [part.strip() for part in line.split(",")]
        if len(parts) < 11:
            raise ValueError(
                f"Row {idx}: expected at least 11 comma-separated columns in course template, got {len(parts)}"
            )

        tail = parts[-9:]
        code = parts[0]
        course_name = ", ".join([x for x in parts[1:-9] if x != ""]).strip()

        raw = {
            "course_code": code,
            "course_name": course_name,
            "lecture_hrs": tail[0],
            "tutorial_hrs": tail[1],
            "practical_hrs": tail[2],
            "credit": tail[3],
            "cia_marks": tail[5],
            "see_marks": tail[6],
            "category": tail[8],
            "count_towards_limit": 1,
        }
        rows.append(normalize_course_row(raw, idx))

    return rows


def normalize_course_row(raw: Dict[str, Any], row_no: int) -> Dict[str, Any]:
    if not isinstance(raw, dict):
        raise ValueError(f"Row {row_no}: each item must be an object")

    def as_int(value: Any, field: str) -> Optional[int]:
        if value is None or value == "":
            return None
        try:
            return int(value)
        except (TypeError, ValueError):
            raise ValueError(f"Row {row_no}: invalid integer for {field}: {value}")

    course_id = as_int(raw.get("course_id"), "course_id")
    course_code = raw.get("course_code")
    if isinstance(course_code, str):
        course_code = course_code.strip()
    if not course_id and not course_code:
        raise ValueError(f"Row {row_no}: provide either course_id or course_code")

    curriculum_id = as_int(raw.get("curriculum_id"), "curriculum_id")
    department_id = as_int(raw.get("department_id"), "department_id")
    count_towards_limit = as_int(raw.get("count_towards_limit"), "count_towards_limit")
    if count_towards_limit is None:
        count_towards_limit = 1
    if count_towards_limit not in (0, 1):
        raise ValueError(f"Row {row_no}: count_towards_limit must be 0 or 1")

    course_name = raw.get("course_name")
    if isinstance(course_name, str):
        course_name = course_name.strip()
    course_type = raw.get("course_type")
    category = raw.get("category")

    def as_int_default(field: str, default: int = 0) -> int:
        value = raw.get(field)
        parsed = as_int(value, field)
        return default if parsed is None else parsed

    lecture_hrs = as_int_default("lecture_hrs", 0)
    tutorial_hrs = as_int_default("tutorial_hrs", 0)
    practical_hrs = as_int_default("practical_hrs", 0)
    activity_hrs = as_int_default("activity_hrs", 0)
    tw_sl_hrs = as_int_default("tw_sl_hrs", 0)
    cia_marks = as_int_default("cia_marks", 0)
    see_marks = as_int_default("see_marks", 0)
    credit = as_int_default("credit", 0)

    theory_total_hrs = as_int(raw.get("theory_total_hrs"), "theory_total_hrs")
    tutorial_total_hrs = as_int(raw.get("tutorial_total_hrs"), "tutorial_total_hrs")
    practical_total_hrs = as_int(raw.get("practical_total_hrs"), "practical_total_hrs")
    activity_total_hrs = as_int(raw.get("activity_total_hrs"), "activity_total_hrs")
    total_hrs = as_int(raw.get("total_hrs"), "total_hrs")

    if theory_total_hrs is None:
        theory_total_hrs = lecture_hrs * 15
    if tutorial_total_hrs is None:
        tutorial_total_hrs = tutorial_hrs * 15
    if practical_total_hrs is None:
        practical_total_hrs = practical_hrs * 15
    if activity_total_hrs is None:
        activity_total_hrs = activity_hrs * 15
    if total_hrs is None:
        total_hrs = theory_total_hrs + tutorial_total_hrs + practical_total_hrs + activity_total_hrs

    return {
        "row_no": row_no,
        "curriculum_id": curriculum_id,
        "course_id": course_id,
        "course_code": course_code,
        "department_id": department_id,
        "count_towards_limit": count_towards_limit,
        "course_name": course_name,
        "course_type": course_type,
        "category": category,
        "credit": credit,
        "lecture_hrs": lecture_hrs,
        "tutorial_hrs": tutorial_hrs,
        "practical_hrs": practical_hrs,
        "activity_hrs": activity_hrs,
        "tw_sl_hrs": tw_sl_hrs,
        "cia_marks": cia_marks,
        "see_marks": see_marks,
        "theory_total_hrs": theory_total_hrs,
        "tutorial_total_hrs": tutorial_total_hrs,
        "practical_total_hrs": practical_total_hrs,
        "activity_total_hrs": activity_total_hrs,
        "total_hrs": total_hrs,
    }


class ApiError(Exception):
    pass


def _make_url(base_url: str, path: str) -> str:
    base = base_url.rstrip("/")
    suffix = path if path.startswith("/") else f"/{path}"
    return f"{base}{suffix}"


def api_request(
    base_url: str,
    method: str,
    path: str,
    token: str,
    payload: Optional[Dict[str, Any]] = None,
) -> Any:
    url = _make_url(base_url, path)
    body = None
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"

    if payload is not None:
        body = json.dumps(payload).encode("utf-8")

    req = request.Request(url=url, data=body, method=method.upper(), headers=headers)
    try:
        with request.urlopen(req, timeout=30) as resp:
            raw = resp.read().decode("utf-8")
            if not raw:
                return None
            try:
                return json.loads(raw)
            except json.JSONDecodeError:
                return raw
    except error.HTTPError as exc:
        detail = ""
        try:
            detail = exc.read().decode("utf-8")
        except Exception:
            detail = str(exc)
        raise ApiError(f"{method.upper()} {path} failed ({exc.code}): {detail}") from exc
    except error.URLError as exc:
        raise ApiError(f"{method.upper()} {path} failed: {exc}") from exc


def get_semesters(base_url: str, token: str, curriculum_id: int) -> List[Dict[str, Any]]:
    data = api_request(base_url, "GET", f"/api/curriculum/{curriculum_id}/semesters", token)
    if isinstance(data, list):
        return data
    raise ApiError(f"Unexpected semesters response for curriculum {curriculum_id}: {data}")


def get_or_create_semester_card(
    base_url: str,
    token: str,
    curriculum_id: int,
    semester_number: int,
    dry_run: bool,
) -> Tuple[Optional[int], bool]:
    semesters = get_semesters(base_url, token, curriculum_id)
    for card in semesters:
        card_type = (card.get("card_type") or "semester").strip().lower()
        if card_type != "semester":
            continue
        if int(card.get("semester_number") or -1) == semester_number:
            return int(card["id"]), False

    if dry_run:
        return None, True

    created = api_request(
        base_url,
        "POST",
        f"/api/curriculum/{curriculum_id}/semester",
        token,
        payload={"semester_number": semester_number, "card_type": "semester"},
    )
    if not isinstance(created, dict) or "id" not in created:
        raise ApiError(f"Unexpected create semester response for curriculum {curriculum_id}: {created}")
    return int(created["id"]), True


def get_semester_courses(
    base_url: str,
    token: str,
    curriculum_id: int,
    semester_id: int,
) -> List[Dict[str, Any]]:
    data = api_request(
        base_url,
        "GET",
        f"/api/curriculum/{curriculum_id}/semester/{semester_id}/courses",
        token,
    )
    if isinstance(data, list):
        return data
    raise ApiError(f"Unexpected semester courses response for curriculum {curriculum_id}: {data}")


def get_course_by_id(base_url: str, token: str, course_id: int) -> Dict[str, Any]:
    data = api_request(base_url, "GET", f"/api/course/{course_id}", token)
    if isinstance(data, dict):
        return data
    raise ApiError(f"Unexpected course response for course_id={course_id}: {data}")


def get_all_departments_semester_courses(
    base_url: str,
    token: str,
    semester_number: int,
) -> List[Dict[str, Any]]:
    data = api_request(base_url, "GET", f"/api/all-departments/semester/{semester_number}/courses", token)
    if isinstance(data, list):
        return data
    raise ApiError(f"Unexpected all-departments response for semester {semester_number}: {data}")


def build_add_course_payload(
    row: Dict[str, Any],
    source_course: Optional[Dict[str, Any]],
    default_course_type: str,
) -> Dict[str, Any]:
    src = source_course or {}

    course_code = row.get("course_code") or src.get("course_code")
    course_name = row.get("course_name") or src.get("course_name")
    course_type = row.get("course_type") if row.get("course_type") is not None else src.get("course_type")
    category = row.get("category") or src.get("category") or ""

    if not course_code:
        raise ValueError(f"Row {row.get('row_no')}: missing course_code")
    if not course_name:
        raise ValueError(
            f"Row {row.get('row_no')}: missing course_name (required when course_id is not resolvable)"
        )
    if course_type is None or course_type == "":
        course_type = default_course_type

    lecture_hrs = row.get("lecture_hrs", src.get("lecture_hrs", 0))
    tutorial_hrs = row.get("tutorial_hrs", src.get("tutorial_hrs", 0))
    practical_hrs = row.get("practical_hrs", src.get("practical_hrs", 0))
    activity_hrs = row.get("activity_hrs", src.get("activity_hrs", 0))

    theory_total_hrs = row.get("theory_total_hrs", lecture_hrs * 15)
    tutorial_total_hrs = row.get("tutorial_total_hrs", tutorial_hrs * 15)
    practical_total_hrs = row.get("practical_total_hrs", practical_hrs * 15)
    activity_total_hrs = row.get("activity_total_hrs", activity_hrs * 15)
    total_hrs = row.get(
        "total_hrs",
        theory_total_hrs + tutorial_total_hrs + practical_total_hrs + activity_total_hrs,
    )

    return {
        "course_code": course_code,
        "course_name": course_name,
        "course_type": course_type,
        "category": category,
        "credit": row.get("credit", src.get("credit", 0)),
        "lecture_hrs": lecture_hrs,
        "tutorial_hrs": tutorial_hrs,
        "practical_hrs": practical_hrs,
        "activity_hrs": activity_hrs,
        "tw_sl_hrs": row.get("tw_sl_hrs", src.get("tw_sl_hrs", src.get("tw_sl", 0))),
        "theory_total_hrs": theory_total_hrs,
        "tutorial_total_hrs": tutorial_total_hrs,
        "practical_total_hrs": practical_total_hrs,
        "activity_total_hrs": activity_total_hrs,
        "total_hrs": total_hrs,
        "cia_marks": row.get("cia_marks", src.get("cia_marks", 0)),
        "see_marks": row.get("see_marks", src.get("see_marks", 0)),
        "count_towards_limit": bool(row.get("count_towards_limit", 1)),
    }


def main() -> int:
    args = parse_args()

    try:
        course_rows = load_courses(args.courses_file)
    except Exception as exc:
        print(f"Input error: {exc}", file=sys.stderr)
        return 1

    if not course_rows:
        print("No course rows found in input file.")
        return 0

    inserted_count = 0
    skipped_existing = 0
    created_semester_cards = 0
    failed_rows = 0
    resolved_course_ids = 0

    # Resolve curriculum_id per row.
    resolved_rows: List[Dict[str, Any]] = []
    for row in course_rows:
        curriculum_id = row.get("curriculum_id") or (args.curriculum_id if args.curriculum_id > 0 else None)
        if not curriculum_id:
            print(
                f"Skipped row {row.get('row_no')}: curriculum_id missing. "
                "Provide row.curriculum_id or --curriculum-id.",
                file=sys.stderr,
            )
            failed_rows += 1
            continue
        out = dict(row)
        out["curriculum_id"] = int(curriculum_id)
        resolved_rows.append(out)

    if not resolved_rows:
        print("No valid rows to process after curriculum resolution.")
        return 1

    if args.resolve_existing_by_semester > 0:
        try:
            api_rows = get_all_departments_semester_courses(
                args.base_url,
                args.auth_token,
                args.resolve_existing_by_semester,
            )
            code_to_id: Dict[str, int] = {}
            for item in api_rows:
                code = str(item.get("course_code") or "").strip().upper()
                cid = item.get("course_id")
                if not code or cid is None:
                    continue
                try:
                    code_to_id[code] = int(cid)
                except (TypeError, ValueError):
                    continue

            for row in resolved_rows:
                if row.get("course_id"):
                    continue
                code = str(row.get("course_code") or "").strip().upper()
                matched_id = code_to_id.get(code)
                if matched_id:
                    row["course_id"] = matched_id
                    resolved_course_ids += 1
        except Exception as exc:
            print(
                f"Warning: could not auto-resolve course IDs from semester "
                f"{args.resolve_existing_by_semester}: {exc}",
                file=sys.stderr,
            )

    rows_by_curriculum: Dict[int, List[Dict[str, Any]]] = defaultdict(list)
    for row in resolved_rows:
        rows_by_curriculum[row["curriculum_id"]].append(row)

    course_cache: Dict[int, Dict[str, Any]] = {}

    print(
        f"Processing {len(rows_by_curriculum)} curriculum(s), "
        f"{len(resolved_rows)} course row(s), semester={args.semester_number}, dry_run={args.dry_run}"
    )

    for curriculum_id in sorted(rows_by_curriculum.keys()):
        try:
            semester_id, was_created = get_or_create_semester_card(
                args.base_url,
                args.auth_token,
                curriculum_id,
                args.semester_number,
                args.dry_run,
            )
        except Exception as exc:
            print(f"Curriculum {curriculum_id}: failed to resolve/create semester card: {exc}", file=sys.stderr)
            failed_rows += len(rows_by_curriculum[curriculum_id])
            continue

        if was_created:
            created_semester_cards += 1

        if semester_id is None:
            print(f"Curriculum {curriculum_id} -> SemesterCard [DRY-RUN new semester card]")
            existing_ids = set()
            existing_codes = set()
        else:
            print(f"Curriculum {curriculum_id} -> SemesterCard {semester_id}")
            try:
                existing = get_semester_courses(args.base_url, args.auth_token, curriculum_id, semester_id)
            except Exception as exc:
                print(
                    f"  Failed to fetch existing semester courses for curriculum {curriculum_id}: {exc}",
                    file=sys.stderr,
                )
                failed_rows += len(rows_by_curriculum[curriculum_id])
                continue
            existing_ids = {int(item.get("course_id")) for item in existing if item.get("course_id") is not None}
            existing_codes = {
                str(item.get("course_code", "")).strip().upper()
                for item in existing
                if item.get("course_code")
            }

        for row in rows_by_curriculum[curriculum_id]:
            course_id = row.get("course_id")
            course_code = (row.get("course_code") or "").strip()
            course_code_norm = course_code.upper()

            if course_id and course_id in existing_ids:
                skipped_existing += 1
                print(f"  Skip existing: course_id={course_id}")
                continue
            if course_code and course_code_norm in existing_codes:
                skipped_existing += 1
                print(f"  Skip existing: course_code={course_code}")
                continue

            if args.dry_run:
                inserted_count += 1
                print(
                    f"  [DRY-RUN] Add course to curriculum={curriculum_id}, semester={args.semester_number}, "
                    f"course_id={course_id}, course_code={course_code or 'N/A'}, "
                    f"count_towards_limit={row.get('count_towards_limit', 1)}"
                )
                continue

            try:
                source_course = None
                if course_id:
                    if course_id not in course_cache:
                        course_cache[course_id] = get_course_by_id(args.base_url, args.auth_token, course_id)
                    source_course = course_cache[course_id]

                payload = build_add_course_payload(row, source_course, args.default_course_type)
                resp = api_request(
                    args.base_url,
                    "POST",
                    f"/api/curriculum/{curriculum_id}/semester/{semester_id}/course",
                    args.auth_token,
                    payload=payload,
                )

                inserted_count += 1
                new_course = resp.get("course") if isinstance(resp, dict) else None
                new_id = new_course.get("id") if isinstance(new_course, dict) else None
                new_code = new_course.get("course_code") if isinstance(new_course, dict) else payload.get("course_code")

                if isinstance(new_id, int):
                    existing_ids.add(new_id)
                if new_code:
                    existing_codes.add(str(new_code).strip().upper())

                print(f"  Inserted: {payload.get('course_code')} (curriculum {curriculum_id})")
            except Exception as exc:
                failed_rows += 1
                print(
                    f"  Failed row {row.get('row_no')} (curriculum={curriculum_id}, "
                    f"course_id={course_id}, course_code={course_code}): {exc}",
                    file=sys.stderr,
                )

    print("\nSummary")
    print(f"  Inserted mappings: {inserted_count}")
    print(f"  Skipped existing mappings: {skipped_existing}")
    print(f"  Auto-resolved course_id by code: {resolved_course_ids}")
    print(f"  Semester cards created (or would create in dry-run): {created_semester_cards}")
    print(f"  Failed rows: {failed_rows}")

    return 0 if failed_rows == 0 else 2


if __name__ == "__main__":
    raise SystemExit(main())
