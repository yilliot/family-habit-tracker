<?php

namespace App\Http\Requests\HabitTracker;

use App\Models\Habit;
use Illuminate\Foundation\Http\FormRequest;

class StoreHabitRequest extends FormRequest
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
            'name' => Habit::nameRules(),
            'display_order' => ['nullable', 'integer', 'min:0'],
        ];
    }
}
