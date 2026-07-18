package missionweaveprotocol

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var protocolRFC3339 = regexp.MustCompile(
	`^([0-9]{4})-([0-9]{2})-([0-9]{2})[Tt]([0-9]{2}):([0-9]{2}):([0-9]{2})(?:\.([0-9]+))?([Zz]|[+-][0-9]{2}:[0-9]{2})$`,
)

// RFC3339Instant represents an RFC 3339 instant without truncating fractional-second precision.
// Its fields are private so copies cannot mutate evidence retained by a verified result.
type RFC3339Instant struct {
	epochSecond int64
	fraction    string
}

// EpochSecond returns whole UTC seconds since 1970-01-01T00:00:00Z.
func (instant RFC3339Instant) EpochSecond() int64 { return instant.epochSecond }

// Fraction returns fractional-second digits with insignificant trailing zeroes removed.
func (instant RFC3339Instant) Fraction() string { return instant.fraction }

func (instant RFC3339Instant) compare(other RFC3339Instant) int {
	if instant.epochSecond < other.epochSecond {
		return -1
	}
	if instant.epochSecond > other.epochSecond {
		return 1
	}
	width := len(instant.fraction)
	if len(other.fraction) > width {
		width = len(other.fraction)
	}
	left := instant.fraction + strings.Repeat("0", width-len(instant.fraction))
	right := other.fraction + strings.Repeat("0", width-len(other.fraction))
	return strings.Compare(left, right)
}

func parseProtocolRFC3339(value string) (RFC3339Instant, error) {
	match := protocolRFC3339.FindStringSubmatch(value)
	if match == nil {
		return RFC3339Instant{}, errors.New("not an RFC 3339 timestamp")
	}
	parts := make([]int, 6)
	for index := range parts {
		parsed, err := strconv.Atoi(match[index+1])
		if err != nil {
			return RFC3339Instant{}, errors.New("timestamp contains an invalid decimal field")
		}
		parts[index] = parsed
	}
	year, month, day := parts[0], parts[1], parts[2]
	hour, minute, second := parts[3], parts[4], parts[5]
	if year == 0 {
		return RFC3339Instant{}, errors.New("year 0000 is not supported")
	}
	if month < 1 || month > 12 {
		return RFC3339Instant{}, errors.New("month is outside 01 through 12")
	}
	monthLengths := [...]int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	if isGregorianLeapYear(year) {
		monthLengths[1] = 29
	}
	if day < 1 || day > monthLengths[month-1] {
		return RFC3339Instant{}, errors.New("day is invalid for the Gregorian month")
	}
	if hour > 23 || minute > 59 {
		return RFC3339Instant{}, errors.New("time is outside 00:00 through 23:59")
	}
	if second > 59 {
		return RFC3339Instant{}, errors.New("leap-second spellings are not supported in v0.1")
	}

	offsetText := match[8]
	if offsetText == "-00:00" {
		return RFC3339Instant{}, errors.New("unknown-local-offset spelling -00:00 is not an instant")
	}
	offsetSeconds := 0
	if offsetText != "Z" && offsetText != "z" {
		offsetHour, _ := strconv.Atoi(offsetText[1:3])
		offsetMinute, _ := strconv.Atoi(offsetText[4:6])
		if offsetHour > 23 || offsetMinute > 59 {
			return RFC3339Instant{}, errors.New("numeric offset is outside RFC 3339 bounds")
		}
		direction := 1
		if offsetText[0] == '-' {
			direction = -1
		}
		offsetSeconds = direction * (offsetHour*3600 + offsetMinute*60)
	}
	localSecond := daysFromCivil(year, month, day)*86400 + int64(hour*3600+minute*60+second)
	return RFC3339Instant{
		epochSecond: localSecond - int64(offsetSeconds),
		fraction:    strings.TrimRight(match[7], "0"),
	}, nil
}

func validateProtocolDateTimeFormat(value any) error {
	text, ok := value.(string)
	if !ok {
		return nil
	}
	if _, err := parseProtocolRFC3339(text); err != nil {
		return fmt.Errorf("invalid RFC 3339 instant: %w", err)
	}
	return nil
}

func isGregorianLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

func daysFromCivil(year, month, day int) int64 {
	adjustedYear := year
	if month <= 2 {
		adjustedYear--
	}
	era := adjustedYear / 400
	yearOfEra := adjustedYear - era*400
	adjustedMonth := month + 9
	if month > 2 {
		adjustedMonth = month - 3
	}
	dayOfYear := (153*adjustedMonth+2)/5 + day - 1
	dayOfEra := yearOfEra*365 + yearOfEra/4 - yearOfEra/100 + dayOfYear
	return int64(era*146097 + dayOfEra - 719468)
}
