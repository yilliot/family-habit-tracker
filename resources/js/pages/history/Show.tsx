import { Head, Link } from '@inertiajs/react';
import ProgressStrip from '@/components/habit-tracker/progress-strip';
import SprintGrid from '@/components/habit-tracker/sprint-grid';
import history from '@/routes/history';
import type { HtHistoryShowProps } from '@/types/habit-tracker';

export default function HistoryShow({
    sprint,
    days,
    participants,
    today,
}: HtHistoryShowProps) {
    return (
        <>
            <Head title="Archived sprint" />
            <div className="space-y-4">
                <Link
                    href={history.index().url}
                    className="ht-eyebrow inline-block hover:text-[var(--ink)]"
                >
                    ← Back to history
                </Link>
                <ProgressStrip participants={participants} />
                <SprintGrid
                    sprint={sprint}
                    days={days}
                    participants={participants}
                    today={today}
                    readOnly
                />
            </div>
        </>
    );
}
