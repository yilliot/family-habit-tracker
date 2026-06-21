<?php

namespace App\Http\Requests\HabitTracker;

use Illuminate\Contracts\Validation\ValidationRule;
use Illuminate\Foundation\Http\FormRequest;

class StoreAdjustmentRequest extends FormRequest
{
    /**
     * Determine if the user is authorized to make this request.
     */
    public function authorize(): bool
    {
        return true;
    }

    /**
     * Get the validation rules that apply to the request.
     *
     * `points` is the positive magnitude to deduct; the controller stores it
     * as a negative adjustment amount.
     *
     * @return array<string, ValidationRule|array<mixed>|string>
     */
    public function rules(): array
    {
        return [
            'points' => ['required', 'integer', 'min:1', 'max:9999'],
            'reason' => ['nullable', 'string', 'max:120'],
        ];
    }
}
