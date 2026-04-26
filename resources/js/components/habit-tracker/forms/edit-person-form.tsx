import { useForm } from '@inertiajs/react';
import type { FormEvent } from 'react';
import { useState } from 'react';
import people from '@/routes/people';
import type { HtPersonWithTrash } from '@/types/habit-tracker';

type Props = {
    person: HtPersonWithTrash;
};

export default function EditPersonForm({ person }: Props) {
    const [editing, setEditing] = useState(false);
    const { data, setData, patch, processing, errors, reset } = useForm({
        name: person.name,
        display_order: person.display_order,
    });

    function submit(e: FormEvent) {
        e.preventDefault();
        patch(people.update(person.id).url, {
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
                disabled={person.deleted_at !== null}
            >
                Edit
            </button>
        );
    }

    return (
        <form
            onSubmit={submit}
            className="flex flex-col gap-2 sm:flex-row sm:items-end"
        >
            <div className="flex-1">
                <label
                    className="ht-label"
                    htmlFor={`edit-person-${person.id}`}
                >
                    Name
                </label>
                <input
                    id={`edit-person-${person.id}`}
                    type="text"
                    className="ht-input"
                    value={data.name}
                    onChange={(e) => setData('name', e.target.value)}
                    maxLength={50}
                    aria-invalid={!!errors.name}
                />
                {errors.name && <p className="ht-error">{errors.name}</p>}
            </div>
            <div className="w-24">
                <label className="ht-label" htmlFor={`edit-order-${person.id}`}>
                    Order
                </label>
                <input
                    id={`edit-order-${person.id}`}
                    type="number"
                    className="ht-input"
                    value={data.display_order}
                    onChange={(e) =>
                        setData('display_order', Number(e.target.value))
                    }
                    aria-invalid={!!errors.display_order}
                />
                {errors.display_order && (
                    <p className="ht-error">{errors.display_order}</p>
                )}
            </div>
            <div className="flex gap-2">
                <button type="submit" className="ht-btn" disabled={processing}>
                    {processing ? 'Saving…' : 'Save'}
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
            </div>
        </form>
    );
}
