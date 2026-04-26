<?php

namespace App\Http\Requests\HabitTracker;

use App\Models\Person;
use Illuminate\Foundation\Http\FormRequest;

class UpdatePersonRequest extends FormRequest
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
            'name' => array_merge(['sometimes'], $this->optionalize(Person::nameRules())),
            'display_order' => array_merge(['sometimes'], Person::displayOrderRules()),
        ];
    }

    /**
     * Convert a "required" rule set into an optional ("nullable") one for
     * partial-update requests.
     *
     * @param  array<int, mixed>  $rules
     * @return array<int, mixed>
     */
    private function optionalize(array $rules): array
    {
        return array_values(array_filter(
            $rules,
            fn ($rule) => $rule !== 'required',
        ));
    }
}
