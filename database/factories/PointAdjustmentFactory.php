<?php

namespace Database\Factories;

use App\Models\PointAdjustment;
use App\Models\SprintParticipant;
use Illuminate\Database\Eloquent\Factories\Factory;

/**
 * @extends Factory<PointAdjustment>
 */
class PointAdjustmentFactory extends Factory
{
    /**
     * Define the model's default state.
     *
     * @return array<string, mixed>
     */
    public function definition(): array
    {
        return [
            'sprint_participant_id' => SprintParticipant::factory(),
            'amount' => -$this->faker->numberBetween(1, 20),
            'reason' => $this->faker->optional()->sentence(3),
            'created_at' => now(),
        ];
    }
}
