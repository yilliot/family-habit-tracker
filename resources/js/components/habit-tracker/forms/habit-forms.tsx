import { useForm } from '@inertiajs/react';
import type { FormEvent } from 'react';
import { useState } from 'react';
import habits from '@/routes/habits';
import type { HtHabit } from '@/types/habit-tracker';

export function AddHabitForm({
    sprintParticipantId,
}: {
    sprintParticipantId: number;
}) {
    const { data, setData, post, processing, errors, reset } = useForm({
        name: '',
    });

    function submit(e: FormEvent) {
        e.preventDefault();
        post(habits.store(sprintParticipantId).url, {
            preserveScroll: true,
            onSuccess: () => reset('name'),
        });
    }

    return (
        <form onSubmit={submit} className="flex items-end gap-2">
            <div className="flex-1">
                <label
                    className="ht-label sr-only"
                    htmlFor={`add-habit-${sprintParticipantId}`}
                >
                    Add habit
                </label>
                <input
                    id={`add-habit-${sprintParticipantId}`}
                    type="text"
                    className="ht-input"
                    value={data.name}
                    onChange={(e) => setData('name', e.target.value)}
                    placeholder="Add a habit"
                    maxLength={80}
                    aria-invalid={!!errors.name}
                />
                {errors.name && <p className="ht-error">{errors.name}</p>}
            </div>
            <button
                type="submit"
                className="ht-btn"
                disabled={processing || data.name.trim().length === 0}
            >
                {processing ? 'Adding…' : 'Add'}
            </button>
        </form>
    );
}

export function EditHabitForm({ habit }: { habit: HtHabit }) {
    const [editing, setEditing] = useState(false);
    const { data, setData, patch, processing, errors, reset } = useForm({
        name: habit.name,
        display_order: habit.display_order,
    });

    function submit(e: FormEvent) {
        e.preventDefault();
        patch(habits.update(habit.id).url, {
            preserveScroll: true,
            onSuccess: () => setEditing(false),
        });
    }

    if (!editing) {
        return (
            <button
                type="button"
                className="ht-btn ht-btn-ghost text-[10px]"
                onClick={() => setEditing(true)}
            >
                Edit
            </button>
        );
    }

    return (
        <form onSubmit={submit} className="flex items-end gap-2">
            <div className="flex-1">
                <input
                    type="text"
                    className="ht-input"
                    value={data.name}
                    onChange={(e) => setData('name', e.target.value)}
                    maxLength={80}
                    aria-invalid={!!errors.name}
                />
                {errors.name && <p className="ht-error">{errors.name}</p>}
            </div>
            <input
                type="number"
                className="ht-input w-16"
                value={data.display_order}
                onChange={(e) =>
                    setData('display_order', Number(e.target.value))
                }
            />
            <button type="submit" className="ht-btn" disabled={processing}>
                Save
            </button>
            <button
                type="button"
                className="ht-btn ht-btn-ghost"
                onClick={() => {
                    reset();
                    setEditing(false);
                }}
            >
                Cancel
            </button>
        </form>
    );
}
