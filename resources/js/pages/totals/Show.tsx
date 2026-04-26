import { Head } from '@inertiajs/react';
import EmptyState from '@/components/habit-tracker/empty-state';
import type { HtTotalsProps } from '@/types/habit-tracker';

export default function TotalsShow({ people }: HtTotalsProps) {
    return (
        <>
            <Head title="Totals" />
            <div className="mx-auto max-w-4xl pt-6">
                <h2
                    className="ht-mast-title mb-6 text-3xl"
                    style={{ fontFamily: 'var(--font-serif-sc)' }}
                >
                    Lifetime totals
                </h2>

                {people.length === 0 ? (
                    <EmptyState
                        title="No data yet"
                        description="First ticks coming soon."
                    />
                ) : (
                    <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
                        {people.map((p) => (
                            <article
                                key={p.person_id}
                                className="ht-progress-card"
                            >
                                <h3
                                    className="ht-mast-title text-2xl"
                                    style={{
                                        fontFamily: 'var(--font-serif-sc)',
                                    }}
                                >
                                    {p.name}
                                </h3>
                                <div className="ht-divider mt-3 grid grid-cols-3 gap-2 pt-3 text-center">
                                    <Stat
                                        big={p.total_ticks_earned}
                                        label="Points"
                                    />
                                    <Stat
                                        big={p.total_rewards_achieved}
                                        label="Rewards"
                                    />
                                    <Stat
                                        big={p.sprints_participated_count}
                                        label="Sprints"
                                    />
                                </div>
                            </article>
                        ))}
                    </div>
                )}
            </div>
        </>
    );
}

function Stat({ big, label }: { big: number; label: string }) {
    return (
        <div>
            <div
                className="ht-num text-3xl"
                style={{ fontFamily: 'var(--font-fraunces)' }}
            >
                {big}
            </div>
            <div className="ht-eyebrow mt-1 text-[var(--ink-3)]">{label}</div>
        </div>
    );
}
