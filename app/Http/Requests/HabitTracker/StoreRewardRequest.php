<?php

namespace App\Http\Requests\HabitTracker;

use App\Models\Reward;
use Illuminate\Foundation\Http\FormRequest;

class StoreRewardRequest extends FormRequest
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
            'name' => Reward::nameRules(),
            'cost' => Reward::costRules(),
        ];
    }
}
