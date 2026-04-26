import { Link, usePage } from '@inertiajs/react';
import { useState } from 'react';
import settings from '@/routes/settings';
import type { HtFlash } from '@/types/habit-tracker';

export default function CelebrationOverlay() {
    const page = usePage<{ flash?: HtFlash }>();
    const achievement = page.props.flash?.achievement ?? null;
    const [dismissedRewardIds, setDismissedRewardIds] = useState<Set<number>>(
        new Set(),
    );

    const open =
        achievement !== null && !dismissedRewardIds.has(achievement.reward_id);

    if (!open || !achievement) {
        return null;
    }

    function dismiss() {
        if (!achievement) {
            return;
        }

        const id = achievement.reward_id;

        setDismissedRewardIds((s) => {
            const next = new Set(s);

            next.add(id);

            return next;
        });
    }

    return (
        <div
            className="ht-celebration-bg fixed inset-0 z-50 flex items-center justify-center px-4"
            role="dialog"
            aria-modal="true"
            aria-labelledby="celebration-title"
        >
            <div className="ht-celebration-card mx-auto max-w-md p-8 text-center">
                <div className="ht-eyebrow text-[var(--terracotta)]">
                    Goal hit · 达成
                </div>
                <h2
                    id="celebration-title"
                    className="ht-mast-title mt-3 text-3xl"
                    style={{ fontFamily: 'var(--font-serif-sc)' }}
                >
                    {achievement.person_name}
                </h2>
                <p
                    className="ht-italic mt-2 text-lg text-[var(--ink-2)]"
                    style={{ fontFamily: 'var(--font-fraunces)' }}
                >
                    earned{' '}
                    <strong className="text-[var(--ink)]">
                        {achievement.reward_name}
                    </strong>
                </p>
                <p className="mt-4 text-sm text-[var(--ink-3)]">
                    Surplus points roll into the next reward you pick.
                </p>
                <div className="mt-6 flex flex-col gap-3 sm:flex-row sm:justify-center">
                    <Link
                        href={`${settings.index().url}?tab=rewards&person=${achievement.person_id}`}
                        className="ht-btn ht-btn-terracotta"
                        onClick={dismiss}
                    >
                        Pick next reward
                    </Link>
                    <button
                        type="button"
                        className="ht-btn ht-btn-ghost"
                        onClick={dismiss}
                    >
                        Maybe later
                    </button>
                </div>
            </div>
        </div>
    );
}
