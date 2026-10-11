"""Report lines: one project, one day, a duration, as the owner wrote them in a summary.

The rules mirror the Go core, which checks every line again when it reads the file
(`internal/activity/infrastructure/file/collector.go` `validLine`, and `kernel.NewDuration`):

- `minutes` is a positive multiple of 15 (`kernel.Granularity`);
- `date` is a Monday-Friday of the ISO week being collected (`WeekID.Contains`);
- `project` is not empty after trimming; it is stored trimmed.

The week check needs the week, so it runs only when validation gets
`context={"week": IsoWeek}`; the extractor always passes it.
"""

from __future__ import annotations

from datetime import date
from typing import Annotated, Literal

from pydantic import (
    BaseModel,
    ConfigDict,
    Field,
    StrictInt,
    StrictStr,
    StringConstraints,
    ValidationInfo,
    field_validator,
    model_validator,
)

from allocation_agent.activity.week import IsoWeek

GRANULARITY_MINUTES = 15
_SLACK_TS = r"^\d+\.\d+$"
_ISO_DATE = r"^\d{4}-\d{2}-\d{2}$"

Project = Annotated[str, StringConstraints(strip_whitespace=True, min_length=1)]


def check_minutes(v: int) -> int:
    if v <= 0 or v % GRANULARITY_MINUTES:
        raise ValueError(f"minutes {v} must be a positive multiple of {GRANULARITY_MINUTES}")
    return v


def check_in_week(d: date, info: ValidationInfo) -> date:
    week = (info.context or {}).get("week")
    if isinstance(week, IsoWeek) and not week.contains_workday(d):
        raise ValueError(f"date {d.isoformat()} is not a Monday-Friday of {week}")
    return d


class ReportLine(BaseModel):
    """One parsed line, as written to `report_lines` in the activity file (contract v2)."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    date: date
    project: Project
    minutes: StrictInt
    notes: StrictStr
    source_ts: str = Field(pattern=_SLACK_TS)
    source_channel_id: str = Field(min_length=1)

    @field_validator("minutes")
    @classmethod
    def _valid_minutes(cls, v: int) -> int:
        return check_minutes(v)

    @field_validator("date")
    @classmethod
    def _in_week(cls, v: date, info: ValidationInfo) -> date:
        return check_in_week(v, info)


class LLMLine(BaseModel):
    """One line exactly as the LLM must return it. Strict: no coercion, no extra keys."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    date: Annotated[StrictStr, Field(pattern=_ISO_DATE)]
    project: Annotated[StrictStr, StringConstraints(strip_whitespace=True, min_length=1)]
    minutes: StrictInt
    notes: StrictStr

    @field_validator("minutes")
    @classmethod
    def _valid_minutes(cls, v: int) -> int:
        return check_minutes(v)

    @field_validator("date")
    @classmethod
    def _real_date(cls, v: str, info: ValidationInfo) -> str:
        try:
            d = date.fromisoformat(v)
        except ValueError:
            raise ValueError(f"date {v!r} is not a calendar date") from None
        check_in_week(d, info)
        return v


class LLMOutput(BaseModel):
    """The whole LLM answer for one message: `{"lines": [...]}`; an empty list is valid."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    lines: list[LLMLine]


class ParserInfo(BaseModel):
    """How `report_lines` were made (contract v2 `parser`)."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    kind: Literal["llm", "manual"]
    provider: str | None = None
    model: str | None = None

    @model_validator(mode="after")
    def _llm_names_its_model(self) -> ParserInfo:
        if self.kind == "llm" and not (self.provider and self.model):
            raise ValueError("parser kind llm needs provider and model")
        return self
