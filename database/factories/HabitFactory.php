<?php

namespace Database\Factories;

use App\Models\Habit;
use App\Models\SprintParticipant;
use Illuminate\Database\Eloquent\Factories\Factory;

/**
 * @extends Factory<Habit>
 */
class HabitFactory extends Factory
{
    protected $model = Habit::class;

    /**
     * @return array<string, mixed>
     */
    public function definition(): array
    {
        return [
            'sprint_participant_id' => SprintParticipant::factory(),
            'name' => fake()->words(2, true),
            'display_order' => 0,
        ];
    }
}
