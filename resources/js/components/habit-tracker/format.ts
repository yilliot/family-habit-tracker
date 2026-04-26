const DOW = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
const DOW_SHORT = ['S', 'M', 'T', 'W', 'T', 'F', 'S'];
const MONTHS = [
    'Jan',
    'Feb',
    'Mar',
    'Apr',
    'May',
    'Jun',
    'Jul',
    'Aug',
    'Sep',
    'Oct',
    'Nov',
    'Dec',
];
const MONTH_LONG = [
    'January',
    'February',
    'March',
    'April',
    'May',
    'June',
    'July',
    'August',
    'September',
    'October',
    'November',
    'December',
];

/**
 * Parses an ISO date string ("YYYY-MM-DD") as a local-date Date object,
 * sidestepping JS's UTC interpretation of date-only strings.
 */
export function parseIsoDate(iso: string): Date {
    const [y, m, d] = iso.split('-').map(Number);

    return new Date(y, (m ?? 1) - 1, d ?? 1);
}

export function formatLong(iso: string): string {
    const d = parseIsoDate(iso);

    return `${DOW[d.getDay()]} · ${MONTHS[d.getMonth()]} ${d.getDate()}, ${d.getFullYear()}`;
}

export function formatRange(startIso: string, endIso: string): string {
    const a = parseIsoDate(startIso);
    const b = parseIsoDate(endIso);
    const sameYear = a.getFullYear() === b.getFullYear();
    const left = `${MONTHS[a.getMonth()]} ${a.getDate()}`;
    const right = sameYear
        ? `${MONTHS[b.getMonth()]} ${b.getDate()}, ${b.getFullYear()}`
        : `${MONTHS[b.getMonth()]} ${b.getDate()}, ${b.getFullYear()}`;

    return `${left} – ${right}`;
}

export function dowShort(iso: string): string {
    return DOW_SHORT[parseIsoDate(iso).getDay()];
}

export function dayNumber(iso: string): number {
    return parseIsoDate(iso).getDate();
}

export function monthLabel(iso: string): string {
    const d = parseIsoDate(iso);

    return `${MONTH_LONG[d.getMonth()]} ${d.getFullYear()}`;
}

export function isSunday(iso: string): boolean {
    return parseIsoDate(iso).getDay() === 0;
}

/**
 * Counts how many sprint sequence number to show in the masthead.
 * For now we fold this into a "No. {id}" formatting — visible in the
 * eyebrow line.
 */
export function sprintSequence(id: number): string {
    return id.toString().padStart(2, '0');
}
