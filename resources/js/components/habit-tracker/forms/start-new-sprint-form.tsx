import { useForm } from '@inertiajs/react';
import type { FormEvent } from 'react';
import { useMemo, useState } from 'react';
import sprints from '@/routes/sprints';
import type { HtNextSprintParticipant, HtPerson } from '@/types/habit-tracker';

type ParticipantState = {
    person_id: number;
    person_name: string;
    included: boolean;
    habits: Array<{ name: string; display_order: number }>;
    reward: {
        mode: 'keep_current' | 'new' | 'none';
        name: string;
        cost: number;
        existing_id?: number;
    };
};

type Props = {
    defaults: {
        start_date: string;
        end_date: string;
        participants: HtNextSprintParticipant[];
    };
    available: HtPerson[];
};

export default function StartNewSprintForm({ defaults, available }: Props) {
    const initialFromDefaults = defaults.participants.map<ParticipantState>(
        (p) => ({
            person_id: p.person_id,
            person_name: p.person_name,
            included: true,
            habits:
                p.habits.length > 0
                    ? p.habits.map((h) => ({
                          name: h.name,
                          display_order: h.display_order,
                      }))
                    : [{ name: '', display_order: 1 }],
            reward: p.active_reward
                ? {
                      mode: 'keep_current',
                      name: p.active_reward.name,
                      cost: p.active_reward.cost,
                      existing_id: p.active_reward.id,
                  }
                : { mode: 'none', name: '', cost: 1 },
        }),
    );

    // Add anyone available but not already in defaults (un-checked by default)
    const defaultIds = new Set(defaults.participants.map((p) => p.person_id));
    const extraFromAvailable = available
        .filter((p) => !defaultIds.has(p.id))
        .map<ParticipantState>((p) => ({
            person_id: p.id,
            person_name: p.name,
            included: false,
            habits: [{ name: '', display_order: 1 }],
            reward: { mode: 'none', name: '', cost: 1 },
        }));

    const [participants, setParticipants] = useState<ParticipantState[]>([
        ...initialFromDefaults,
        ...extraFromAvailable,
    ]);

    const { data, setData, post, processing, errors, transform } = useForm<{
        start_date: string;
        end_date: string;
        participants: Array<{
            person_id: number;
            habits: Array<{ name: string; display_order: number }>;
            reward: {
                mode: 'keep_current' | 'new' | 'none';
                name?: string;
                cost?: number;
            } | null;
        }>;
    }>({
        start_date: defaults.start_date,
        end_date: defaults.end_date,
        participants: [],
    });

    transform((d) => ({
        ...d,
        participants: participants
            .filter((p) => p.included)
            .map((p) => ({
                person_id: p.person_id,
                habits: p.habits
                    .filter((h) => h.name.trim().length > 0)
                    .map((h, idx) => ({
                        name: h.name,
                        display_order: h.display_order || idx + 1,
                    })),
                reward:
                    p.reward.mode === 'new'
                        ? {
                              mode: 'new',
                              name: p.reward.name,
                              cost: p.reward.cost,
                          }
                        : { mode: p.reward.mode },
            })),
    }));

    const includedCount = useMemo(
        () => participants.filter((p) => p.included).length,
        [participants],
    );

    function updateParticipant(idx: number, patch: Partial<ParticipantState>) {
        setParticipants((current) =>
            current.map((p, i) => (i === idx ? { ...p, ...patch } : p)),
        );
    }

    function updateHabit(
        pIdx: number,
        hIdx: number,
        patch: Partial<{ name: string; display_order: number }>,
    ) {
        setParticipants((current) =>
            current.map((p, i) =>
                i === pIdx
                    ? {
                          ...p,
                          habits: p.habits.map((h, j) =>
                              j === hIdx ? { ...h, ...patch } : h,
                          ),
                      }
                    : p,
            ),
        );
    }

    function addHabit(pIdx: number) {
        setParticipants((current) =>
            current.map((p, i) =>
                i === pIdx
                    ? {
                          ...p,
                          habits: [
                              ...p.habits,
                              { name: '', display_order: p.habits.length + 1 },
                          ],
                      }
                    : p,
            ),
        );
    }

    function removeHabit(pIdx: number, hIdx: number) {
        setParticipants((current) =>
            current.map((p, i) =>
                i === pIdx
                    ? {
                          ...p,
                          habits: p.habits.filter((_, j) => j !== hIdx),
                      }
                    : p,
            ),
        );
    }

    function submit(e: FormEvent) {
        e.preventDefault();
        post(sprints.store().url, { preserveScroll: true });
    }

    return (
        <form onSubmit={submit} className="space-y-8">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div>
                    <label className="ht-label" htmlFor="sprint-start">
                        Start date
                    </label>
                    <input
                        id="sprint-start"
                        type="date"
                        className="ht-input"
                        value={data.start_date}
                        onChange={(e) => setData('start_date', e.target.value)}
                        aria-invalid={!!errors.start_date}
                    />
                    {errors.start_date && (
                        <p className="ht-error">{errors.start_date}</p>
                    )}
                </div>
                <div>
                    <label className="ht-label" htmlFor="sprint-end">
                        End date
                    </label>
                    <input
                        id="sprint-end"
                        type="date"
                        className="ht-input"
                        value={data.end_date}
                        onChange={(e) => setData('end_date', e.target.value)}
                        aria-invalid={!!errors.end_date}
                    />
                    {errors.end_date && (
                        <p className="ht-error">{errors.end_date}</p>
                    )}
                </div>
            </div>

            <div className="space-y-6">
                <h3 className="ht-eyebrow font-semibold text-[var(--ink)]">
                    Participants
                </h3>
                {participants.length === 0 && (
                    <p className="text-sm text-[var(--ink-3)]">
                        Add a family member first, then come back here to start
                        a sprint.
                    </p>
                )}
                {participants.map((p, pIdx) => (
                    <div
                        key={p.person_id}
                        className="ht-divider space-y-4 pt-4"
                    >
                        <label className="flex items-center gap-2">
                            <input
                                type="checkbox"
                                checked={p.included}
                                onChange={(e) =>
                                    updateParticipant(pIdx, {
                                        included: e.target.checked,
                                    })
                                }
                            />
                            <span
                                className="ht-mast-title text-lg"
                                style={{ fontFamily: 'var(--font-serif-sc)' }}
                            >
                                {p.person_name}
                            </span>
                        </label>

                        {p.included && (
                            <>
                                <div>
                                    <h4 className="ht-label">Habits</h4>
                                    <div className="space-y-2">
                                        {p.habits.map((h, hIdx) => (
                                            <div
                                                key={hIdx}
                                                className="flex items-center gap-2"
                                            >
                                                <input
                                                    type="text"
                                                    className="ht-input flex-1"
                                                    value={h.name}
                                                    onChange={(e) =>
                                                        updateHabit(
                                                            pIdx,
                                                            hIdx,
                                                            {
                                                                name: e.target
                                                                    .value,
                                                            },
                                                        )
                                                    }
                                                    placeholder="e.g. 30 min reading"
                                                    maxLength={80}
                                                />
                                                <button
                                                    type="button"
                                                    className="ht-btn ht-btn-ghost text-[10px]"
                                                    onClick={() =>
                                                        removeHabit(pIdx, hIdx)
                                                    }
                                                    disabled={
                                                        p.habits.length <= 1
                                                    }
                                                >
                                                    Remove
                                                </button>
                                            </div>
                                        ))}
                                    </div>
                                    <button
                                        type="button"
                                        className="ht-btn ht-btn-ghost mt-2 text-[10px]"
                                        onClick={() => addHabit(pIdx)}
                                    >
                                        + Add habit
                                    </button>
                                </div>

                                <div>
                                    <h4 className="ht-label">Reward</h4>
                                    <div className="space-y-2">
                                        {p.reward.existing_id !== undefined && (
                                            <label className="flex items-center gap-2 text-sm">
                                                <input
                                                    type="radio"
                                                    name={`reward-mode-${pIdx}`}
                                                    checked={
                                                        p.reward.mode ===
                                                        'keep_current'
                                                    }
                                                    onChange={() =>
                                                        updateParticipant(
                                                            pIdx,
                                                            {
                                                                reward: {
                                                                    ...p.reward,
                                                                    mode: 'keep_current',
                                                                },
                                                            },
                                                        )
                                                    }
                                                />
                                                Keep current ·{' '}
                                                <span className="ht-italic">
                                                    {p.reward.name} ·{' '}
                                                    {p.reward.cost}pt
                                                </span>
                                            </label>
                                        )}
                                        <label className="flex items-center gap-2 text-sm">
                                            <input
                                                type="radio"
                                                name={`reward-mode-${pIdx}`}
                                                checked={
                                                    p.reward.mode === 'new'
                                                }
                                                onChange={() =>
                                                    updateParticipant(pIdx, {
                                                        reward: {
                                                            ...p.reward,
                                                            mode: 'new',
                                                        },
                                                    })
                                                }
                                            />
                                            New reward
                                        </label>
                                        <label className="flex items-center gap-2 text-sm">
                                            <input
                                                type="radio"
                                                name={`reward-mode-${pIdx}`}
                                                checked={
                                                    p.reward.mode === 'none'
                                                }
                                                onChange={() =>
                                                    updateParticipant(pIdx, {
                                                        reward: {
                                                            ...p.reward,
                                                            mode: 'none',
                                                        },
                                                    })
                                                }
                                            />
                                            No reward yet
                                        </label>
                                        {p.reward.mode === 'new' && (
                                            <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
                                                <input
                                                    type="text"
                                                    className="ht-input sm:col-span-2"
                                                    placeholder="Reward name"
                                                    value={p.reward.name}
                                                    maxLength={80}
                                                    onChange={(e) =>
                                                        updateParticipant(
                                                            pIdx,
                                                            {
                                                                reward: {
                                                                    ...p.reward,
                                                                    name: e
                                                                        .target
                                                                        .value,
                                                                },
                                                            },
                                                        )
                                                    }
                                                />
                                                <input
                                                    type="number"
                                                    className="ht-input"
                                                    placeholder="Cost (1-9999)"
                                                    min={1}
                                                    max={9999}
                                                    value={p.reward.cost}
                                                    onChange={(e) =>
                                                        updateParticipant(
                                                            pIdx,
                                                            {
                                                                reward: {
                                                                    ...p.reward,
                                                                    cost: Number(
                                                                        e.target
                                                                            .value,
                                                                    ),
                                                                },
                                                            },
                                                        )
                                                    }
                                                />
                                            </div>
                                        )}
                                    </div>
                                </div>
                            </>
                        )}
                    </div>
                ))}
            </div>

            <button
                type="submit"
                className="ht-btn ht-btn-terracotta"
                disabled={processing || includedCount === 0}
            >
                {processing ? 'Starting…' : 'Start sprint'}
            </button>
        </form>
    );
}
