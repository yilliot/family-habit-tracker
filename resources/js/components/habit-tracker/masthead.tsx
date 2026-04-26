import { usePage } from '@inertiajs/react';
import { formatLong, sprintSequence } from '@/components/habit-tracker/format';
import TopNav from '@/components/habit-tracker/top-nav';
import type { HtSprintMeta } from '@/types/habit-tracker';

type MastheadPageProps = {
    sprint?: HtSprintMeta | null;
    active_sprint?: { sprint: HtSprintMeta } | null;
};

export default function Masthead() {
    const page = usePage<MastheadPageProps>();
    const sprint =
        page.props.sprint ?? page.props.active_sprint?.sprint ?? null;
    const isArchived = sprint?.status === 'archived';

    return (
        <header className="ht-masthead grid grid-cols-1 gap-6 pb-6 md:grid-cols-[1fr_auto_1fr] md:items-end md:gap-8">
            <div>
                <div className="ht-eyebrow mb-2">
                    The Family Ledger
                    {sprint ? ` · No. ${sprintSequence(sprint.id)}` : ''}
                </div>
                <TopNav />
            </div>

            <div className="text-center">
                <h1 className="ht-mast-title text-[44px] leading-none md:text-[60px] lg:text-[72px]">
                    家庭习惯表
                </h1>
                <div className="ht-italic mt-2 text-[16px] text-[var(--ink-2)] md:text-[20px]">
                    A Family Habit Tracker · one small tick at a time
                </div>
            </div>

            <div className="md:text-right">
                <div className="inline-block text-left">
                    {sprint ? (
                        <>
                            <div className="ht-eyebrow text-[10px] leading-7 md:text-[11px]">
                                <span className="font-semibold text-[var(--ink)]">
                                    Begins
                                </span>{' '}
                                &nbsp; {formatLong(sprint.start_date)}
                            </div>
                            <div className="ht-eyebrow text-[10px] leading-7 md:text-[11px]">
                                <span className="font-semibold text-[var(--ink)]">
                                    Ends
                                </span>{' '}
                                &nbsp;&nbsp;&nbsp; {formatLong(sprint.end_date)}
                            </div>
                            {isArchived && (
                                <div className="ht-italic mt-1 text-[12px] text-[var(--terracotta)]">
                                    Archived
                                </div>
                            )}
                        </>
                    ) : (
                        <div className="ht-italic text-[14px] text-[var(--ink-2)]">
                            No active sprint
                        </div>
                    )}
                </div>
            </div>
        </header>
    );
}
