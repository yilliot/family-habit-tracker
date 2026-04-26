<?php

namespace App\Actions\HabitTracker;

use App\Models\Sprint;
use Illuminate\Validation\ValidationException;

class EndSprint
{
    /**
     * Archive the given sprint. After this point all child writes are
     * rejected by the other Habit-Tracker actions.
     */
    public function execute(Sprint $sprint): Sprint
    {
        if ($sprint->isArchived()) {
            throw ValidationException::withMessages([
                'sprint' => 'Sprint is already archived.',
            ]);
        }

        $sprint->update([
            'status' => Sprint::STATUS_ARCHIVED,
            'ended_at' => now(),
        ]);

        return $sprint->refresh();
    }
}
