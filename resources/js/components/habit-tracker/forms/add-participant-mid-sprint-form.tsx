import { useForm } from '@inertiajs/react';
import type { FormEvent } from 'react';
import { useState } from 'react';
import sprintParticipants from '@/routes/sprint-participants';
import type { HtPerson } from '@/types/habit-tracker';

type Props = {
    sprintId: number;
    available: HtPerson[];
};

export default function AddParticipantMidSprintForm({
    sprintId,
    available,
}: Props) {
    const [habitInputs, setHabitInputs] = useState<string[]>(['']);
    const { data, setData, post, processing, errors, reset, transform } =
        useForm<{
            person_id: number | null;
            habits: Array<{ name: string }>;
            reward: { mode: 'new' | 'none'; name?: string; cost?: number };
        }>({
            person_id: null,
            habits: [],
            reward: { mode: 'none' },
        });

    transform((d) => ({
        ...d,
        habits: habitInputs
            .filter((h) => h.trim().length > 0)
            .map((name) => ({ name })),
    }));

    function submit(e: FormEvent) {
        e.preventDefault();

        if (!data.person_id) {
            return;
        }

        post(sprintParticipants.store(sprintId).url, {
            preserveScroll: true,
            onSuccess: () => {
                reset();
                setHabitInputs(['']);
            },
        });
    }

    if (available.length === 0) {
        return (
            <p className="text-sm text-[var(--ink-3)]">
                Everyone is already on this sprint.
            </p>
        );
    }

    return (
        <form onSubmit={submit} className="space-y-3">
            <div>
                <label className="ht-label" htmlFor="add-participant-person">
                    Person
                </label>
                <select
                    id="add-participant-person"
                    className="ht-input"
                    value={data.person_id ?? ''}
                    onChange={(e) =>
                        setData(
                            'person_id',
                            e.target.value ? Number(e.target.value) : null,
                        )
                    }
                    aria-invalid={!!errors.person_id}
                >
                    <option value="">— Choose —</option>
                    {available.map((p) => (
                        <option key={p.id} value={p.id}>
                            {p.name}
                        </option>
                    ))}
                </select>
                {errors.person_id && (
                    <p className="ht-error">{errors.person_id}</p>
                )}
            </div>

            <div>
                <label className="ht-label">Habits (optional)</label>
                <div className="space-y-2">
                    {habitInputs.map((h, idx) => (
                        <input
                            key={idx}
                            type="text"
                            className="ht-input"
                            value={h}
                            placeholder="Habit name"
                            maxLength={80}
                            onChange={(e) => {
                                const next = [...habitInputs];

                                next[idx] = e.target.value;
                                setHabitInputs(next);
                            }}
                        />
                    ))}
                </div>
                <button
                    type="button"
                    className="ht-btn ht-btn-ghost mt-2 text-[10px]"
                    onClick={() => setHabitInputs((h) => [...h, ''])}
                >
                    + Add habit
                </button>
            </div>

            <div>
                <label className="ht-label">Reward (optional)</label>
                <div className="space-y-2">
                    <label className="flex items-center gap-2 text-sm">
                        <input
                            type="radio"
                            name="add-participant-reward-mode"
                            checked={data.reward.mode === 'none'}
                            onChange={() => setData('reward', { mode: 'none' })}
                        />
                        No reward yet
                    </label>
                    <label className="flex items-center gap-2 text-sm">
                        <input
                            type="radio"
                            name="add-participant-reward-mode"
                            checked={data.reward.mode === 'new'}
                            onChange={() =>
                                setData('reward', {
                                    mode: 'new',
                                    name: '',
                                    cost: 50,
                                })
                            }
                        />
                        New reward
                    </label>
                    {data.reward.mode === 'new' && (
                        <div className="grid grid-cols-1 gap-2 sm:grid-cols-[1fr_120px]">
                            <input
                                type="text"
                                className="ht-input"
                                placeholder="Reward name"
                                value={data.reward.name ?? ''}
                                maxLength={80}
                                onChange={(e) =>
                                    setData('reward', {
                                        ...data.reward,
                                        name: e.target.value,
                                    })
                                }
                            />
                            <input
                                type="number"
                                className="ht-input"
                                placeholder="Cost"
                                min={1}
                                max={9999}
                                value={data.reward.cost ?? 50}
                                onChange={(e) =>
                                    setData('reward', {
                                        ...data.reward,
                                        cost: Number(e.target.value),
                                    })
                                }
                            />
                        </div>
                    )}
                </div>
            </div>

            <button
                type="submit"
                className="ht-btn"
                disabled={processing || !data.person_id}
            >
                {processing ? 'Adding…' : 'Add participant'}
            </button>
        </form>
    );
}
