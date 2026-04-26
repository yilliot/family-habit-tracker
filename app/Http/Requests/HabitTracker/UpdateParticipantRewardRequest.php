<?php

namespace App\Http\Requests\HabitTracker;

use Illuminate\Foundation\Http\FormRequest;
use Illuminate\Validation\Rule;

class UpdateParticipantRewardRequest extends FormRequest
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
            'reward_id' => [
                'required',
                'integer',
                Rule::exists('rewards', 'id'),
            ],
        ];
    }
}
