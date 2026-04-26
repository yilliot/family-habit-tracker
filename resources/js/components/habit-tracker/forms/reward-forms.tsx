import { useForm } from '@inertiajs/react';
import type { FormEvent } from 'react';
import { useState } from 'react';
import rewards from '@/routes/rewards';
import type { HtReward } from '@/types/habit-tracker';

export function AddRewardForm({ personId }: { personId: number }) {
    const { data, setData, post, processing, errors, reset } = useForm({
        name: '',
        cost: 50,
    });

    function submit(e: FormEvent) {
        e.preventDefault();
        post(rewards.store(personId).url, {
            preserveScroll: true,
            onSuccess: () => reset(),
        });
    }

    return (
        <form
            onSubmit={submit}
            className="grid grid-cols-1 gap-2 sm:grid-cols-[1fr_120px_auto]"
        >
            <div>
                <label className="ht-label" htmlFor={`reward-name-${personId}`}>
                    Reward name
                </label>
                <input
                    id={`reward-name-${personId}`}
                    type="text"
                    className="ht-input"
                    value={data.name}
                    onChange={(e) => setData('name', e.target.value)}
                    placeholder="e.g. Lego Minecraft"
                    maxLength={80}
                    aria-invalid={!!errors.name}
                />
                {errors.name && <p className="ht-error">{errors.name}</p>}
            </div>
            <div>
                <label className="ht-label" htmlFor={`reward-cost-${personId}`}>
                    Cost (pts)
                </label>
                <input
                    id={`reward-cost-${personId}`}
                    type="number"
                    className="ht-input"
                    min={1}
                    max={9999}
                    value={data.cost}
                    onChange={(e) => setData('cost', Number(e.target.value))}
                    aria-invalid={!!errors.cost}
                />
                {errors.cost && <p className="ht-error">{errors.cost}</p>}
            </div>
            <button
                type="submit"
                className="ht-btn self-end"
                disabled={processing || data.name.trim().length === 0}
            >
                {processing ? 'Saving…' : 'Add reward'}
            </button>
        </form>
    );
}

export function EditRewardForm({ reward }: { reward: HtReward }) {
    const [editing, setEditing] = useState(false);
    const { data, setData, patch, processing, errors, reset } = useForm({
        name: reward.name,
        cost: reward.cost,
    });
    const isAchieved = reward.achieved_at !== null;

    function submit(e: FormEvent) {
        e.preventDefault();
        patch(rewards.update(reward.id).url, {
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
        <form
            onSubmit={submit}
            className="grid grid-cols-1 gap-2 sm:grid-cols-[1fr_120px_auto_auto]"
        >
            <div>
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
            <div>
                <input
                    type="number"
                    className="ht-input"
                    min={1}
                    max={9999}
                    value={data.cost}
                    onChange={(e) => setData('cost', Number(e.target.value))}
                    disabled={isAchieved}
                    aria-invalid={!!errors.cost}
                />
                {errors.cost && <p className="ht-error">{errors.cost}</p>}
            </div>
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
