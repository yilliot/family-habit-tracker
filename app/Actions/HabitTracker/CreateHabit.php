<?php

namespace App\Actions\HabitTracker;

use App\Models\Habit;
use App\Models\SprintParticipant;
use Illuminate\Validation\ValidationException;

class CreateHabit
{
    public function execute(SprintParticipant $participant, string $name, ?int $displayOrder = null): Habit
    {
        $participant->loadMissing('sprint');

        if ($participant->sprint->isArchived()) {
            throw ValidationException::withMessages([
                'sprint' => 'Cannot add habits to an archived sprint.',
            ]);
        }

        return $participant->habits()->create([
            'name' => $name,
            'display_order' => $displayOrder ?? ((int) $participant->habits()->max('display_order') + 1),
        ]);
    }
}
