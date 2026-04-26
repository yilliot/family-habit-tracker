import { Link } from '@inertiajs/react';
import settings from '@/routes/settings';
import type { HtParticipant } from '@/types/habit-tracker';

type Props = {
    participants: HtParticipant[];
};

export default function ProgressStrip({ participants }: Props) {
    if (participants.length === 0) {
        return null;
    }

    return (
        <section
            aria-label="Progress toward each person's reward"
            className="ht-progress-strip"
        >
            {participants.map((p) => {
                const reward = p.active_reward;
                const balance = p.points_balance;
                const cost = reward?.cost ?? 0;
                const pct =
                    reward && reward.cost > 0
                        ? Math.min(
                              100,
                              Math.round((balance / reward.cost) * 100),
                          )
                        : 0;

                return (
                    <article key={p.id} className="ht-progress-card">
                        <div className="flex items-baseline justify-between gap-2">
                            <h3
                                className="ht-mast-title text-[20px] leading-tight"
                                style={{ fontFamily: 'var(--font-serif-sc)' }}
                            >
                                {p.person.name}
                            </h3>
                            <div className="ht-italic text-[12px] text-[var(--ink-3)]">
                                {pct}%
                            </div>
                        </div>

                        <div className="mt-1 flex items-baseline gap-2">
                            <span
                                className="ht-num text-[24px]"
                                style={{ fontFamily: 'var(--font-fraunces)' }}
                            >
                                {balance}
                            </span>
                            {reward ? (
                                <>
                                    <span className="ht-italic text-[14px] text-[var(--ink-3)]">
                                        / {cost}
                                    </span>
                                    <span className="ht-italic ml-2 text-[13px] text-[var(--ink-2)]">
                                        {reward.name}
                                    </span>
                                </>
                            ) : (
                                <Link
                                    href={settings.index().url}
                                    className="ht-italic ml-2 text-[13px] text-[var(--terracotta)] underline-offset-2 hover:underline"
                                >
                                    Pick a reward
                                </Link>
                            )}
                        </div>

                        <div className="ht-progress-track mt-3">
                            <div
                                className="ht-progress-fill"
                                style={{ width: `${pct}%` }}
                            />
                        </div>
                    </article>
                );
            })}
        </section>
    );
}
