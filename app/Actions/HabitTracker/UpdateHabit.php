<?php

namespace App\Actions\HabitTracker;

use App\Models\Habit;
use Illuminate\Validation\ValidationException;

class UpdateHabit
{
    /**
     * @param  array<string, mixed>  $attributes
     */
    public function execute(Habit $habit, array $attributes): Habit
    {
        $habit->loadMissing('sprintParticipant.sprint');

        if ($habit->sprintParticipant->sprint->isArchived()) {
            throw ValidationException::withMessages([
                'sprint' => 'Cannot edit habits inside an archived sprint.',
            ]);
        }

        $habit->fill(array_intersect_key($attributes, array_flip(['name', 'display_order'])));
        $habit->save();

        return $habit->refresh();
    }
}
