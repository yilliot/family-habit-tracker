<?php

namespace App\Http\Requests\HabitTracker;

use App\Models\Habit;
use App\Models\Reward;
use Illuminate\Foundation\Http\FormRequest;
use Illuminate\Validation\Rule;

class StoreSprintRequest extends FormRequest
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
            'start_date' => ['required', 'date'],
            'end_date' => ['required', 'date', 'after_or_equal:start_date'],
            'participants' => ['required', 'array', 'min:1'],
            'participants.*.person_id' => [
                'required',
                'integer',
                Rule::exists('people', 'id')->whereNull('deleted_at'),
            ],
            'participants.*.habits' => ['present', 'array'],
            'participants.*.habits.*.name' => Habit::nameRules(),
            'participants.*.habits.*.display_order' => ['nullable', 'integer', 'min:0'],
            'participants.*.reward' => ['nullable', 'array'],
            'participants.*.reward.mode' => ['required_with:participants.*.reward', 'string', 'in:keep_current,new,none'],
            'participants.*.reward.name' => array_merge(
                ['required_if:participants.*.reward.mode,new', 'nullable'],
                $this->stripRequired(Reward::nameRules()),
            ),
            'participants.*.reward.cost' => array_merge(
                ['required_if:participants.*.reward.mode,new', 'nullable'],
                $this->stripRequired(Reward::costRules()),
            ),
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
