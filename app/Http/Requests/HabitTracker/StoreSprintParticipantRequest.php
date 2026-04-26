<?php

namespace App\Http\Requests\HabitTracker;

use App\Models\Habit;
use App\Models\Reward;
use Illuminate\Foundation\Http\FormRequest;
use Illuminate\Validation\Rule;

class StoreSprintParticipantRequest extends FormRequest
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
            'person_id' => [
                'required',
                'integer',
                Rule::exists('people', 'id')->whereNull('deleted_at'),
            ],
            'habits' => ['nullable', 'array'],
            'habits.*.name' => Habit::nameRules(),
            'habits.*.display_order' => ['nullable', 'integer', 'min:0'],
            'reward' => ['nullable', 'array'],
            'reward.mode' => ['required_with:reward', 'string', 'in:keep_current,new,none'],
            'reward.name' => array_merge(
                ['required_if:reward.mode,new', 'nullable'],
                $this->stripRequired(Reward::nameRules()),
            ),
            'reward.cost' => array_merge(
                ['required_if:reward.mode,new', 'nullable'],
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
