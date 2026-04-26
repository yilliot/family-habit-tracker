import { Head, usePage } from '@inertiajs/react';
import { useMemo, useState } from 'react';
import ConfirmButton from '@/components/habit-tracker/confirm-button';
import EmptyState from '@/components/habit-tracker/empty-state';
import EndSprintConfirm from '@/components/habit-tracker/end-sprint-confirm';
import { formatLong } from '@/components/habit-tracker/format';
import AddParticipantMidSprintForm from '@/components/habit-tracker/forms/add-participant-mid-sprint-form';
import AddPersonForm from '@/components/habit-tracker/forms/add-person-form';
import EditPersonForm from '@/components/habit-tracker/forms/edit-person-form';
import {
    AddHabitForm,
    EditHabitForm,
} from '@/components/habit-tracker/forms/habit-forms';
import {
    AddRewardForm,
    EditRewardForm,
} from '@/components/habit-tracker/forms/reward-forms';
import StartNewSprintForm from '@/components/habit-tracker/forms/start-new-sprint-form';
import SwapActiveRewardSelect from '@/components/habit-tracker/forms/swap-active-reward-select';
import habits from '@/routes/habits';
import people from '@/routes/people';
import rewards from '@/routes/rewards';
import type { HtSettingsProps } from '@/types/habit-tracker';

type Tab = 'people' | 'sprint' | 'rewards';

export default function SettingsIndex({
    people: peopleList,
    active_sprint,
    available_for_sprint,
    rewards_by_person,
    next_sprint_defaults,
}: HtSettingsProps) {
    const page = usePage();
    const queryString = page.url.split('?')[1] ?? '';
    const initialTab = useMemo<Tab>(() => {
        const params = new URLSearchParams(queryString);
        const t = params.get('tab');

        if (t === 'people' || t === 'sprint' || t === 'rewards') {
            return t;
        }

        return 'sprint';
    }, [queryString]);

    const focusedPersonId = useMemo<number | null>(() => {
        const params = new URLSearchParams(queryString);
        const v = params.get('person');

        return v ? Number(v) : null;
    }, [queryString]);

    // Use queryString as the key so navigating with ?tab=… resets state.
    return (
        <SettingsBody
            key={queryString}
            initialTab={initialTab}
            focusedPersonId={focusedPersonId}
            peopleList={peopleList}
            active_sprint={active_sprint}
            available_for_sprint={available_for_sprint}
            rewards_by_person={rewards_by_person}
            next_sprint_defaults={next_sprint_defaults}
        />
    );
}

function SettingsBody({
    initialTab,
    focusedPersonId,
    peopleList,
    active_sprint,
    available_for_sprint,
    rewards_by_person,
    next_sprint_defaults,
}: {
    initialTab: Tab;
    focusedPersonId: number | null;
    peopleList: HtSettingsProps['people'];
    active_sprint: HtSettingsProps['active_sprint'];
    available_for_sprint: HtSettingsProps['available_for_sprint'];
    rewards_by_person: HtSettingsProps['rewards_by_person'];
    next_sprint_defaults: HtSettingsProps['next_sprint_defaults'];
}) {
    const [tab, setTab] = useState<Tab>(initialTab);

    return (
        <>
            <Head title="Settings" />
            <div className="mx-auto max-w-5xl pt-4">
                <div className="ht-divider-strong flex flex-wrap gap-2 pb-0">
                    <button
                        type="button"
                        className="ht-tab"
                        data-active={tab === 'sprint'}
                        onClick={() => setTab('sprint')}
                    >
                        Current sprint
                    </button>
                    <button
                        type="button"
                        className="ht-tab"
                        data-active={tab === 'people'}
                        onClick={() => setTab('people')}
                    >
                        People
                    </button>
                    <button
                        type="button"
                        className="ht-tab"
                        data-active={tab === 'rewards'}
                        onClick={() => setTab('rewards')}
                    >
                        Rewards
                    </button>
                </div>

                <div className="ht-divider-strong border-t pt-6">
                    {tab === 'sprint' && (
                        <SprintTab
                            activeSprint={active_sprint}
                            available={available_for_sprint}
                            nextDefaults={next_sprint_defaults}
                            rewardsByPerson={rewards_by_person}
                        />
                    )}
                    {tab === 'people' && <PeopleTab people={peopleList} />}
                    {tab === 'rewards' && (
                        <RewardsTab
                            people={peopleList}
                            rewardsByPerson={rewards_by_person}
                            focusedPersonId={focusedPersonId}
                        />
                    )}
                </div>
            </div>
        </>
    );
}

function PeopleTab({
    people: peopleList,
}: {
    people: HtSettingsProps['people'];
}) {
    return (
        <section className="space-y-6">
            <h2
                className="ht-mast-title text-2xl"
                style={{ fontFamily: 'var(--font-serif-sc)' }}
            >
                Family
            </h2>
            <AddPersonForm />

            <ul className="ht-divider divide-y divide-[var(--line-3)] pt-4">
                {peopleList.length === 0 ? (
                    <li className="text-sm text-[var(--ink-3)]">
                        Add your first family member above.
                    </li>
                ) : (
                    peopleList.map((p) => (
                        <li
                            key={p.id}
                            className="flex flex-col gap-3 py-3 sm:flex-row sm:items-center sm:justify-between"
                        >
                            <div>
                                <span
                                    className="ht-mast-title text-xl"
                                    style={{
                                        fontFamily: 'var(--font-serif-sc)',
                                    }}
                                >
                                    {p.name}
                                </span>
                                {p.deleted_at && (
                                    <span className="ht-italic ml-2 text-xs text-[var(--ink-3)]">
                                        archived
                                    </span>
                                )}
                                <span className="ht-italic ml-2 text-xs text-[var(--ink-3)]">
                                    order {p.display_order}
                                </span>
                            </div>
                            <div className="flex flex-wrap items-center gap-2">
                                <EditPersonForm person={p} />
                                {!p.deleted_at && (
                                    <ConfirmButton
                                        label="Remove"
                                        confirmTitle="Remove this person?"
                                        confirmBody={`Remove ${p.name}? Their history is preserved, and they can be added to a future sprint.`}
                                        confirmCta="Remove"
                                        method="delete"
                                        url={people.destroy(p.id).url}
                                        danger
                                    />
                                )}
                            </div>
                        </li>
                    ))
                )}
            </ul>
        </section>
    );
}

function SprintTab({
    activeSprint,
    available,
    nextDefaults,
    rewardsByPerson,
}: {
    activeSprint: HtSettingsProps['active_sprint'];
    available: HtSettingsProps['available_for_sprint'];
    nextDefaults: HtSettingsProps['next_sprint_defaults'];
    rewardsByPerson: HtSettingsProps['rewards_by_person'];
}) {
    if (!activeSprint) {
        return (
            <section className="space-y-6">
                <h2
                    className="ht-mast-title text-2xl"
                    style={{ fontFamily: 'var(--font-serif-sc)' }}
                >
                    Start a new sprint
                </h2>
                {nextDefaults.participants.length === 0 &&
                available.length === 0 ? (
                    <EmptyState
                        title="Add a person first"
                        description="You need at least one family member before you can start a sprint."
                    />
                ) : (
                    <StartNewSprintForm
                        defaults={nextDefaults}
                        available={available}
                    />
                )}
            </section>
        );
    }

    const sprint = activeSprint.sprint;

    return (
        <section className="space-y-8">
            <header className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                <div>
                    <h2
                        className="ht-mast-title text-2xl"
                        style={{ fontFamily: 'var(--font-serif-sc)' }}
                    >
                        Active sprint
                    </h2>
                    <p className="ht-italic text-sm text-[var(--ink-2)]">
                        {formatLong(sprint.start_date)} →{' '}
                        {formatLong(sprint.end_date)}
                    </p>
                </div>
                <EndSprintConfirm
                    sprintId={sprint.id}
                    startDate={sprint.start_date}
                    endDate={sprint.end_date}
                />
            </header>

            <div className="space-y-6">
                {activeSprint.participants.map((p) => (
                    <article key={p.id} className="ht-progress-card space-y-3">
                        <h3
                            className="ht-mast-title text-xl"
                            style={{ fontFamily: 'var(--font-serif-sc)' }}
                        >
                            {p.person.name}
                        </h3>
                        <SwapActiveRewardSelect
                            sprintParticipantId={p.id}
                            activeRewardId={p.active_reward?.id ?? null}
                            rewards={rewardsByPerson[p.person.id] ?? []}
                        />

                        <div>
                            <h4 className="ht-label">Habits</h4>
                            <ul className="ht-divider divide-y divide-[var(--line-3)]">
                                {p.habits.length === 0 && (
                                    <li className="ht-italic py-2 text-xs text-[var(--ink-3)]">
                                        No habits yet.
                                    </li>
                                )}
                                {p.habits.map((h) => (
                                    <li
                                        key={h.id}
                                        className="flex flex-col gap-2 py-2 sm:flex-row sm:items-center sm:justify-between"
                                    >
                                        <span className="ht-habit-name">
                                            {h.name}
                                        </span>
                                        <div className="flex gap-2">
                                            <EditHabitForm habit={h} />
                                            <ConfirmButton
                                                label="Remove"
                                                confirmTitle="Remove this habit?"
                                                confirmBody="Existing ticks stay in totals; the habit just disappears from the active grid."
                                                confirmCta="Remove"
                                                method="delete"
                                                url={habits.destroy(h.id).url}
                                                danger
                                            />
                                        </div>
                                    </li>
                                ))}
                            </ul>
                            <div className="ht-divider mt-2 pt-3">
                                <AddHabitForm sprintParticipantId={p.id} />
                            </div>
                        </div>
                    </article>
                ))}
            </div>

            <div className="ht-divider-strong space-y-4 border-t pt-6">
                <h3
                    className="ht-mast-title text-xl"
                    style={{ fontFamily: 'var(--font-serif-sc)' }}
                >
                    Add participant
                </h3>
                <AddParticipantMidSprintForm
                    sprintId={sprint.id}
                    available={available}
                />
            </div>
        </section>
    );
}

function RewardsTab({
    people: peopleList,
    rewardsByPerson,
    focusedPersonId,
}: {
    people: HtSettingsProps['people'];
    rewardsByPerson: HtSettingsProps['rewards_by_person'];
    focusedPersonId: number | null;
}) {
    return (
        <section className="space-y-6">
            <h2
                className="ht-mast-title text-2xl"
                style={{ fontFamily: 'var(--font-serif-sc)' }}
            >
                Rewards
            </h2>
            {peopleList
                .filter((p) => !p.deleted_at)
                .map((p) => {
                    const list = rewardsByPerson[p.id] ?? [];
                    const hasActive = list.some((r) => r.achieved_at === null);
                    const focused = focusedPersonId === p.id;

                    return (
                        <article
                            key={p.id}
                            id={`person-rewards-${p.id}`}
                            className={`ht-progress-card space-y-3 ${focused ? 'ring-2 ring-[var(--terracotta)]' : ''}`}
                        >
                            <h3
                                className="ht-mast-title text-xl"
                                style={{
                                    fontFamily: 'var(--font-serif-sc)',
                                }}
                            >
                                {p.name}
                            </h3>

                            {list.length === 0 ? (
                                <p className="ht-italic text-xs text-[var(--ink-3)]">
                                    No rewards yet — add one below.
                                </p>
                            ) : (
                                <ul className="ht-divider divide-y divide-[var(--line-3)]">
                                    {list.map((r) => (
                                        <li
                                            key={r.id}
                                            className="flex flex-col gap-2 py-2 sm:flex-row sm:items-center sm:justify-between"
                                        >
                                            <div>
                                                <span
                                                    className="ht-mast-title text-base"
                                                    style={{
                                                        fontFamily:
                                                            'var(--font-serif-sc)',
                                                    }}
                                                >
                                                    {r.name}
                                                </span>
                                                <span className="ht-italic ml-2 text-xs text-[var(--ink-3)]">
                                                    {r.cost}pt
                                                    {r.achieved_at
                                                        ? ' · achieved'
                                                        : ' · active'}
                                                </span>
                                            </div>
                                            <div className="flex gap-2">
                                                <EditRewardForm reward={r} />
                                                {r.achieved_at === null && (
                                                    <ConfirmButton
                                                        label="Delete"
                                                        confirmTitle="Delete this reward?"
                                                        confirmBody="Only unachieved rewards can be deleted."
                                                        confirmCta="Delete"
                                                        method="delete"
                                                        url={
                                                            rewards.destroy(
                                                                r.id,
                                                            ).url
                                                        }
                                                        danger
                                                    />
                                                )}
                                            </div>
                                        </li>
                                    ))}
                                </ul>
                            )}

                            {!hasActive && (
                                <div className="ht-divider pt-3">
                                    <AddRewardForm personId={p.id} />
                                </div>
                            )}
                        </article>
                    );
                })}
        </section>
    );
}
