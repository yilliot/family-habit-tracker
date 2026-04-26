<?php

namespace App\Actions\HabitTracker;

use App\Models\Person;
use App\Models\Reward;
use Illuminate\Validation\ValidationException;

class CreateReward
{
    /**
     * Create a new reward for a person. Rejects if the person already has
     * an active (un-achieved) reward.
     */
    public function execute(Person $person, string $name, int $cost): Reward
    {
        if ($person->rewards()->active()->exists()) {
            throw ValidationException::withMessages([
                'reward' => 'This person already has an active reward.',
            ]);
        }

        return Reward::create([
            'person_id' => $person->id,
            'name' => $name,
            'cost' => $cost,
        ]);
    }
}
