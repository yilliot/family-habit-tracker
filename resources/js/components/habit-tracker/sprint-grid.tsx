import { router } from '@inertiajs/react';
import { useCallback, useMemo, useState } from 'react';
import { toast } from 'sonner';
import {
    dayNumber,
    dowShort,
    isSunday,
    monthLabel,
    parseIsoDate,
} from '@/components/habit-tracker/format';
import DeductPointsButton from '@/components/habit-tracker/forms/deduct-points-button';
import ticks from '@/routes/ticks';
import type {
    HtHabit,
    HtParticipant,
    HtSprintDay,
    HtSprintMeta,
} from '@/types/habit-tracker';

type Props = {
    sprint: HtSprintMeta;
    days: HtSprintDay[];
    participants: HtParticipant[];
    today: string;
    readOnly?: boolean;
};

type MonthGroup = {
    label: string;
    span: number;
};

/**
 * Optimistic tick state: tracks per (habitId, date) the locally-toggled
 * value. After the server confirms via Inertia partial reload of
 * `participants`, the prop reflects truth and we drop the local override.
 */
type LocalTickMap = Record<string, boolean>;

const tickKey = (habitId: number, date: string) => `${habitId}|${date}`;

function buildMonthGroups(days: HtSprintDay[]): MonthGroup[] {
    if (days.length === 0) {
        return [];
    }

    const groups: MonthGroup[] = [];
    let i = 0;

    while (i < days.length) {
        const startMonth = parseIsoDate(days[i].date).getMonth();
        let j = i;

        while (
            j < days.length &&
            parseIsoDate(days[j].date).getMonth() === startMonth
        ) {
            j++;
        }

        groups.push({ label: monthLabel(days[i].date), span: j - i });
        i = j;
    }

    return groups;
}

export default function SprintGrid({
    sprint,
    days,
    participants,
    today,
    readOnly = false,
}: Props) {
    const monthGroups = useMemo(() => buildMonthGroups(days), [days]);
    const [localTicks, setLocalTicks] = useState<LocalTickMap>({});
    const [pendingTicks, setPendingTicks] = useState<Set<string>>(new Set());

    const ticksByHabit = useMemo(() => {
        const map: Record<number, Set<string>> = {};

        for (const p of participants) {
            for (const h of p.habits) {
                map[h.id] = new Set(h.tick_dates);
            }
        }

        return map;
    }, [participants]);

    const isTicked = useCallback(
        (habit: HtHabit, date: string) => {
            const key = tickKey(habit.id, date);

            if (key in localTicks) {
                return localTicks[key];
            }

            return ticksByHabit[habit.id]?.has(date) ?? false;
        },
        [localTicks, ticksByHabit],
    );

    const toggle = useCallback(
        (habit: HtHabit, date: string) => {
            if (readOnly) {
                return;
            }

            const key = tickKey(habit.id, date);

            // already in flight — skip duplicate
            if (pendingTicks.has(key)) {
                return;
            }

            const previous = ticksByHabit[habit.id]?.has(date) ?? false;
            const overrideState =
                key in localTicks ? localTicks[key] : previous;
            const optimistic = !overrideState;

            setLocalTicks((m) => ({ ...m, [key]: optimistic }));
            setPendingTicks((s) => {
                const next = new Set(s);

                next.add(key);

                return next;
            });

            router.post(
                ticks.toggle().url,
                { habit_id: habit.id, tick_date: date },
                {
                    preserveScroll: true,
                    preserveState: true,
                    only: ['participants', 'flash'],
                    onError: (errors) => {
                        // Revert optimistic flip
                        setLocalTicks((m) => {
                            const next = { ...m };

                            delete next[key];

                            return next;
                        });
                        const firstError =
                            (Object.values(errors)[0] as string | undefined) ??
                            'Could not save tick';

                        toast.error(firstError);
                    },
                    onSuccess: () => {
                        // Drop local override; props are now truth.
                        setLocalTicks((m) => {
                            const next = { ...m };

                            delete next[key];

                            return next;
                        });
                    },
                    onFinish: () => {
                        setPendingTicks((s) => {
                            const next = new Set(s);

                            next.delete(key);

                            return next;
                        });
                    },
                },
            );
        },
        [localTicks, pendingTicks, readOnly, ticksByHabit],
    );

    const todayIdx = days.findIndex((d) => d.date === today);

    return (
        <div className="-mx-2 overflow-x-auto md:mx-0">
            <table className="ht-grid">
                <colgroup>
                    <col style={{ width: 96 }} />
                    <col style={{ width: 220 }} />
                    {days.map((d) => (
                        <col key={d.date} style={{ width: 44 }} />
                    ))}
                    <col style={{ width: 220 }} />
                </colgroup>
                <thead>
                    <tr className="ht-month-row">
                        <th colSpan={2} />
                        {monthGroups.map((g, idx) => (
                            <th
                                key={`${g.label}-${idx}`}
                                colSpan={g.span}
                                className="ht-month-named"
                            >
                                {g.label}
                            </th>
                        ))}
                        <th />
                    </tr>
                    <tr className="ht-date-row">
                        <th className="ht-corner">Person</th>
                        <th className="ht-corner">Habit</th>
                        {days.map((d, idx) => (
                            <th
                                key={d.date}
                                className={[
                                    'ht-day-head',
                                    d.is_weekend ? 'weekend' : '',
                                    d.is_month_start ? 'month-start' : '',
                                    idx === todayIdx ? 'today' : '',
                                ]
                                    .filter(Boolean)
                                    .join(' ')}
                            >
                                <span className="dow">{dowShort(d.date)}</span>
                                <span className="dnum">
                                    {dayNumber(d.date)}
                                </span>
                            </th>
                        ))}
                        <th className="ht-reward-head">Reward · 奖励</th>
                    </tr>
                </thead>
                <tbody>
                    {participants.map((p) => {
                        const habits = p.habits;

                        if (habits.length === 0) {
                            return (
                                <tr
                                    key={`p-${p.id}`}
                                    className="person-start person-end"
                                >
                                    <td className="person">
                                        <span className="ht-person-name">
                                            {p.person.name}
                                        </span>
                                        <span className="ht-person-balance">
                                            {p.points_balance}
                                            {p.active_reward
                                                ? ` / ${p.active_reward.cost}`
                                                : ' pts'}
                                        </span>
                                        {!readOnly && (
                                            <DeductPointsButton
                                                sprintParticipantId={p.id}
                                                personName={p.person.name}
                                            />
                                        )}
                                    </td>
                                    <td
                                        className="habit"
                                        colSpan={days.length}
                                        style={{
                                            color: 'var(--ink-3)',
                                            fontFamily: 'var(--font-fraunces)',
                                            fontStyle: 'italic',
                                        }}
                                    >
                                        No habits yet — add some in Settings.
                                    </td>
                                    <RewardCell participant={p} />
                                </tr>
                            );
                        }

                        return habits.map((habit, hi) => {
                            const isFirst = hi === 0;
                            const isLast = hi === habits.length - 1;
                            const trClass = [
                                isFirst ? 'person-start' : '',
                                isLast ? 'person-end' : '',
                            ]
                                .filter(Boolean)
                                .join(' ');

                            return (
                                <tr
                                    key={`p-${p.id}-h-${habit.id}`}
                                    className={trClass}
                                >
                                    {isFirst && (
                                        <td
                                            className="person"
                                            rowSpan={habits.length}
                                        >
                                            <span className="ht-person-name">
                                                {p.person.name}
                                            </span>
                                            <span className="ht-person-balance">
                                                {p.points_balance}
                                                {p.active_reward
                                                    ? ` / ${p.active_reward.cost}`
                                                    : ' pts'}
                                            </span>
                                            {!readOnly && (
                                                <DeductPointsButton
                                                    sprintParticipantId={p.id}
                                                    personName={p.person.name}
                                                />
                                            )}
                                        </td>
                                    )}
                                    <td className="habit">
                                        <span className="ht-habit-num">
                                            {hi + 1}.
                                        </span>
                                        <span className="ht-habit-name">
                                            {habit.name}
                                        </span>
                                    </td>
                                    {days.map((d, idx) => {
                                        const isFuture =
                                            d.date > today ||
                                            d.date > sprint.end_date;
                                        const isInRange =
                                            d.date >= sprint.start_date &&
                                            d.date <= sprint.end_date;
                                        const ticked = isTicked(habit, d.date);
                                        const sun = isSunday(d.date);
                                        const isToday = idx === todayIdx;
                                        const tdClasses = [
                                            'day',
                                            d.is_weekend ? 'weekend' : '',
                                            d.is_month_start
                                                ? 'month-start'
                                                : '',
                                            sun ? 'sun' : '',
                                            isToday ? 'today-col' : '',
                                        ]
                                            .filter(Boolean)
                                            .join(' ');
                                        const disabled =
                                            readOnly || isFuture || !isInRange;

                                        return (
                                            <td
                                                key={d.date}
                                                className={tdClasses}
                                            >
                                                <button
                                                    type="button"
                                                    className="ht-cell-btn"
                                                    onClick={() =>
                                                        toggle(habit, d.date)
                                                    }
                                                    disabled={disabled}
                                                    data-ticked={ticked}
                                                    data-future={isFuture}
                                                    data-readonly={
                                                        readOnly || !isInRange
                                                    }
                                                    aria-pressed={ticked}
                                                    aria-label={`${habit.name} on ${d.date}: ${ticked ? 'done' : 'not done'}`}
                                                />
                                            </td>
                                        );
                                    })}
                                    {isFirst && (
                                        <RewardCell
                                            participant={p}
                                            rowSpan={habits.length}
                                        />
                                    )}
                                </tr>
                            );
                        });
                    })}
                </tbody>
            </table>
        </div>
    );
}

function RewardCell({
    participant,
    rowSpan,
}: {
    participant: HtParticipant;
    rowSpan?: number;
}) {
    const reward = participant.active_reward;
    const balance = participant.points_balance;
    const pct =
        reward && reward.cost > 0
            ? Math.min(100, Math.round((balance / reward.cost) * 100))
            : 0;

    return (
        <td className="reward reward-top" rowSpan={rowSpan}>
            <div className="flex h-full flex-col justify-between gap-3 py-1">
                {reward ? (
                    <>
                        <div>
                            <div
                                className="ht-italic text-[11px] tracking-widest text-[var(--ink-3)] uppercase"
                                style={{ letterSpacing: '0.18em' }}
                            >
                                Reward
                            </div>
                            <div
                                className="ht-mast-title mt-1 text-[18px] leading-tight"
                                style={{ fontFamily: 'var(--font-serif-sc)' }}
                            >
                                {reward.name}
                            </div>
                            <div
                                className="ht-italic mt-1 text-[12px] text-[var(--ink-2)]"
                                style={{ fontFamily: 'var(--font-fraunces)' }}
                            >
                                {balance} / {reward.cost} pts
                                {reward.achieved_at ? ' · achieved' : ''}
                            </div>
                        </div>
                        <div className="ht-progress-track">
                            <div
                                className="ht-progress-fill"
                                style={{ width: `${pct}%` }}
                            />
                        </div>
                    </>
                ) : (
                    <div className="ht-italic text-[12px] text-[var(--ink-3)]">
                        No reward set — points accrue as surplus.
                    </div>
                )}
            </div>
        </td>
    );
}
