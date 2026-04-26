import { Head, Link } from '@inertiajs/react';
import CelebrationOverlay from '@/components/habit-tracker/celebration-overlay';
import EmptyState from '@/components/habit-tracker/empty-state';
import ProgressStrip from '@/components/habit-tracker/progress-strip';
import SprintGrid from '@/components/habit-tracker/sprint-grid';
import settings from '@/routes/settings';
import type { HtTrackerProps } from '@/types/habit-tracker';

export default function TrackerShow({
    sprint,
    days,
    participants,
    today,
}: HtTrackerProps) {
    return (
        <>
            <Head title="Tracker" />

            {!sprint || participants.length === 0 ? (
                <EmptyState
                    title="Start your first sprint"
                    description="A sprint is a date range your family commits to a set of habits. Open Settings to configure people, habits, and rewards."
                >
                    <Link
                        href={settings.index().url}
                        className="ht-btn ht-btn-terracotta"
                    >
                        Go to Settings
                    </Link>
                </EmptyState>
            ) : (
                <div className="space-y-6">
                    <ProgressStrip participants={participants} />
                    <SprintGrid
                        sprint={sprint}
                        days={days}
                        participants={participants}
                        today={today}
                    />
                </div>
            )}

            <CelebrationOverlay />
        </>
    );
}
