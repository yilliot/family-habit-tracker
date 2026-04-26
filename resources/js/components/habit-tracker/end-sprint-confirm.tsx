import { router } from '@inertiajs/react';
import { useState } from 'react';
import { formatLong } from '@/components/habit-tracker/format';
import sprints from '@/routes/sprints';

type Props = {
    sprintId: number;
    startDate: string;
    endDate: string;
};

export default function EndSprintConfirm({
    sprintId,
    startDate,
    endDate,
}: Props) {
    const [open, setOpen] = useState(false);
    const [submitting, setSubmitting] = useState(false);

    function endSprint() {
        setSubmitting(true);
        router.post(
            sprints.end(sprintId).url,
            {},
            {
                onFinish: () => {
                    setSubmitting(false);
                    setOpen(false);
                },
            },
        );
    }

    return (
        <>
            <button
                type="button"
                className="ht-btn ht-btn-terracotta"
                onClick={() => setOpen(true)}
            >
                End sprint
            </button>
            {open && (
                <div
                    className="ht-celebration-bg fixed inset-0 z-50 flex items-center justify-center px-4"
                    role="dialog"
                    aria-modal="true"
                    aria-labelledby="end-sprint-title"
                >
                    <div className="ht-celebration-card mx-auto max-w-md p-8">
                        <h2
                            id="end-sprint-title"
                            className="ht-mast-title text-2xl"
                            style={{ fontFamily: 'var(--font-serif-sc)' }}
                        >
                            End the current sprint?
                        </h2>
                        <p className="mt-3 text-sm text-[var(--ink-2)]">
                            End the sprint dated {formatLong(startDate)} —{' '}
                            {formatLong(endDate)}? Sprint becomes read-only.
                            Unspent points will carry to your next sprint.
                        </p>
                        <div className="mt-6 flex flex-col gap-3 sm:flex-row sm:justify-end">
                            <button
                                type="button"
                                className="ht-btn ht-btn-ghost"
                                onClick={() => setOpen(false)}
                                disabled={submitting}
                            >
                                Cancel
                            </button>
                            <button
                                type="button"
                                className="ht-btn ht-btn-terracotta"
                                onClick={endSprint}
                                disabled={submitting}
                            >
                                {submitting ? 'Ending…' : 'End sprint'}
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </>
    );
}
