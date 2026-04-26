<?php

namespace App\Http\Requests\HabitTracker;

use App\Models\Habit;
use Illuminate\Foundation\Http\FormRequest;

class UpdateHabitRequest extends FormRequest
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
            'name' => array_merge(['sometimes'], $this->stripRequired(Habit::nameRules())),
            'display_order' => ['sometimes', 'nullable', 'integer', 'min:0'],
        ];
    }

    /**
     * @param  array<int, mixed>  $rules
     * @return array<int, mixed>
     */
    private function stripRequired(array $rules): array
    {
        return array_values(array_filter($rules, fn ($rule) => $rule !== 'required'));
    }
}
