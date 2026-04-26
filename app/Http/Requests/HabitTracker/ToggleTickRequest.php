<?php

namespace App\Http\Requests\HabitTracker;

use Illuminate\Foundation\Http\FormRequest;
use Illuminate\Validation\Rule;

class ToggleTickRequest extends FormRequest
{
    public function authorize(): bool
    {
        return true;
    }

    /**
     * @return array<string, array<int, mixed>>
     */
    public function rules(): array
    {
        return [
            'habit_id' => [
                'required',
                'integer',
                Rule::exists('habits', 'id')->whereNull('deleted_at'),
            ],
            'tick_date' => ['required', 'date'],
        ];
    }
}
