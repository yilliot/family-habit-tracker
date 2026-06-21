import { useForm } from '@inertiajs/react';
import type { FormEvent } from 'react';
import { useState } from 'react';
import { toast } from 'sonner';
import adjustments from '@/routes/adjustments';

type Props = {
    sprintParticipantId: number;
    personName: string;
};

export default function DeductPointsButton({
    sprintParticipantId,
    personName,
}: Props) {
    const [open, setOpen] = useState(false);
    const { data, setData, post, processing, errors, reset, clearErrors } =
        useForm({
            points: 1,
            reason: '',
        });

    function close() {
        reset();
        clearErrors();
        setOpen(false);
    }

    function submit(e: FormEvent) {
        e.preventDefault();
        post(adjustments.store(sprintParticipantId).url, {
            preserveScroll: true,
            only: ['participants', 'flash'],
            onSuccess: () => {
                toast.success(`Deducted ${data.points} pt from ${personName}`);
                close();
            },
            onError: (errs) => {
                const first =
                    (Object.values(errs)[0] as string | undefined) ??
                    'Could not deduct points';

                toast.error(first);
            },
        });
    }

    return (
        <>
            <button
                type="button"
                className="mt-1 inline-flex items-center text-[10px] tracking-wide text-[var(--ink-3)] underline-offset-2 transition-colors hover:text-[var(--terracotta)] hover:underline"
                onClick={() => setOpen(true)}
                aria-label={`Deduct points from ${personName}`}
            >
                − deduct
            </button>
            {open && (
                <div
                    className="ht-celebration-bg fixed inset-0 z-50 flex items-center justify-center px-4"
                    role="dialog"
                    aria-modal="true"
                >
                    <form
                        onSubmit={submit}
                        className="ht-celebration-card mx-auto w-full max-w-md space-y-4 p-8"
                    >
                        <h3
                            className="ht-mast-title text-xl"
                            style={{ fontFamily: 'var(--font-serif-sc)' }}
                        >
                            Deduct points · {personName}
                        </h3>
                        <div>
                            <label
                                className="ht-label"
                                htmlFor={`deduct-points-${sprintParticipantId}`}
                            >
                                Points to deduct
                            </label>
                            <input
                                id={`deduct-points-${sprintParticipantId}`}
                                type="number"
                                className="ht-input"
                                min={1}
                                max={9999}
                                value={data.points}
                                onChange={(e) =>
                                    setData('points', Number(e.target.value))
                                }
                                aria-invalid={!!errors.points}
                                autoFocus
                            />
                            {errors.points && (
                                <p className="ht-error">{errors.points}</p>
                            )}
                        </div>
                        <div>
                            <label
                                className="ht-label"
                                htmlFor={`deduct-reason-${sprintParticipantId}`}
                            >
                                Reason (optional)
                            </label>
                            <input
                                id={`deduct-reason-${sprintParticipantId}`}
                                type="text"
                                className="ht-input"
                                value={data.reason}
                                onChange={(e) =>
                                    setData('reason', e.target.value)
                                }
                                placeholder="e.g. missed bedtime"
                                maxLength={120}
                                aria-invalid={!!errors.reason}
                            />
                            {errors.reason && (
                                <p className="ht-error">{errors.reason}</p>
                            )}
                        </div>
                        <div className="flex flex-col gap-3 sm:flex-row sm:justify-end">
                            <button
                                type="button"
                                className="ht-btn ht-btn-ghost"
                                onClick={close}
                                disabled={processing}
                            >
                                Cancel
                            </button>
                            <button
                                type="submit"
                                className="ht-btn ht-btn-danger"
                                disabled={processing || data.points < 1}
                            >
                                {processing ? 'Working…' : 'Deduct'}
                            </button>
                        </div>
                    </form>
                </div>
            )}
        </>
    );
}
