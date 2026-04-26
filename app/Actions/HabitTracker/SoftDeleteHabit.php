<?php

namespace App\Actions\HabitTracker;

use App\Models\Habit;
use Illuminate\Validation\ValidationException;

class SoftDeleteHabit
{
    public function execute(Habit $habit): Habit
    {
        $habit->loadMissing('sprintParticipant.sprint');

        if ($habit->sprintParticipant->sprint->isArchived()) {
            throw ValidationException::withMessages([
                'sprint' => 'Cannot delete habits inside an archived sprint.',
            ]);
        }

        $habit->delete();

        return $habit;
    }
}
