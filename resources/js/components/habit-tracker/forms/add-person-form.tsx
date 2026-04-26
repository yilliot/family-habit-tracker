import { useForm } from '@inertiajs/react';
import type { FormEvent } from 'react';
import people from '@/routes/people';

export default function AddPersonForm() {
    const { data, setData, post, processing, errors, reset } = useForm({
        name: '',
    });

    function submit(e: FormEvent) {
        e.preventDefault();
        post(people.store().url, {
            preserveScroll: true,
            onSuccess: () => reset('name'),
        });
    }

    return (
        <form
            onSubmit={submit}
            className="flex flex-col gap-3 sm:flex-row sm:items-end"
        >
            <div className="flex-1">
                <label className="ht-label" htmlFor="add-person-name">
                    New family member
                </label>
                <input
                    id="add-person-name"
                    type="text"
                    className="ht-input"
                    value={data.name}
                    onChange={(e) => setData('name', e.target.value)}
                    maxLength={50}
                    aria-invalid={!!errors.name}
                    placeholder="Name"
                    required
                />
                {errors.name && <p className="ht-error">{errors.name}</p>}
            </div>
            <button
                type="submit"
                className="ht-btn"
                disabled={processing || data.name.trim().length === 0}
            >
                {processing ? 'Adding…' : 'Add person'}
            </button>
        </form>
    );
}
