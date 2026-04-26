import { router } from '@inertiajs/react';
import { useState } from 'react';

type Props = {
    label: string;
    confirmTitle: string;
    confirmBody: string;
    confirmCta?: string;
    method: 'post' | 'patch' | 'delete' | 'put';
    url: string;
    className?: string;
    danger?: boolean;
};

export default function ConfirmButton({
    label,
    confirmTitle,
    confirmBody,
    confirmCta = 'Confirm',
    method,
    url,
    className = '',
    danger = false,
}: Props) {
    const [open, setOpen] = useState(false);
    const [submitting, setSubmitting] = useState(false);

    function submit() {
        setSubmitting(true);
        router[method](
            url,
            {},
            {
                preserveScroll: true,
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
                className={`ht-btn ${danger ? 'ht-btn-danger' : 'ht-btn-ghost'} ${className}`}
                onClick={() => setOpen(true)}
            >
                {label}
            </button>
            {open && (
                <div
                    className="ht-celebration-bg fixed inset-0 z-50 flex items-center justify-center px-4"
                    role="dialog"
                    aria-modal="true"
                >
                    <div className="ht-celebration-card mx-auto max-w-md p-8">
                        <h3
                            className="ht-mast-title text-xl"
                            style={{ fontFamily: 'var(--font-serif-sc)' }}
                        >
                            {confirmTitle}
                        </h3>
                        <p className="mt-3 text-sm text-[var(--ink-2)]">
                            {confirmBody}
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
                                className={`ht-btn ${danger ? 'ht-btn-danger' : 'ht-btn-terracotta'}`}
                                onClick={submit}
                                disabled={submitting}
                            >
                                {submitting ? 'Working…' : confirmCta}
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </>
    );
}
