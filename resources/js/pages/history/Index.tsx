import { Head, Link } from '@inertiajs/react';
import EmptyState from '@/components/habit-tracker/empty-state';
import { formatRange } from '@/components/habit-tracker/format';
import history from '@/routes/history';
import tracker from '@/routes/tracker';
import type { HtHistoryIndexProps } from '@/types/habit-tracker';

export default function HistoryIndex({ sprints }: HtHistoryIndexProps) {
    return (
        <>
            <Head title="History" />
            <div className="mx-auto max-w-3xl pt-6">
                <h2
                    className="ht-mast-title mb-4 text-3xl"
                    style={{ fontFamily: 'var(--font-serif-sc)' }}
                >
                    Past sprints
                </h2>

                {sprints.length === 0 ? (
                    <EmptyState
                        title="No archived sprints yet"
                        description="Your first sprint is in progress — finish it from Settings to see it here."
                    >
                        <Link
                            href={tracker.show().url}
                            className="ht-btn ht-btn-terracotta"
                        >
                            Back to tracker
                        </Link>
                    </EmptyState>
                ) : (
                    <ul className="space-y-3">
                        {sprints.map((s) => (
                            <li
                                key={s.id}
                                className="ht-progress-card flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between"
                            >
                                <div>
                                    <div
                                        className="ht-mast-title text-xl"
                                        style={{
                                            fontFamily: 'var(--font-serif-sc)',
                                        }}
                                    >
                                        {formatRange(s.start_date, s.end_date)}
                                    </div>
                                    <div className="ht-italic text-[12px] text-[var(--ink-3)]">
                                        {s.participant_count} participant
                                        {s.participant_count === 1
                                            ? ''
                                            : 's'} · {s.achievers.length} reward
                                        {s.achievers.length === 1
                                            ? ''
                                            : 's'}{' '}
                                        achieved
                                    </div>
                                    {s.achievers.length > 0 && (
                                        <div className="mt-1 text-xs text-[var(--ink-2)]">
                                            {s.achievers
                                                .map(
                                                    (a) =>
                                                        `${a.name} → ${a.reward_name}`,
                                                )
                                                .join(' · ')}
                                        </div>
                                    )}
                                </div>
                                <Link
                                    href={history.show(s.id).url}
                                    className="ht-btn ht-btn-ghost"
                                >
                                    View
                                </Link>
                            </li>
                        ))}
                    </ul>
                )}
            </div>
        </>
    );
}
