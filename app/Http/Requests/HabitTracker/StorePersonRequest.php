<?php

namespace App\Http\Requests\HabitTracker;

use App\Models\Person;
use Illuminate\Foundation\Http\FormRequest;

class StorePersonRequest extends FormRequest
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
            'name' => Person::nameRules(),
            'display_order' => Person::displayOrderRules(),
        ];
    }
}
