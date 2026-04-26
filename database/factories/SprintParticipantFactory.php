<?php

namespace Database\Factories;

use App\Models\Person;
use App\Models\Sprint;
use App\Models\SprintParticipant;
use Illuminate\Database\Eloquent\Factories\Factory;

/**
 * @extends Factory<SprintParticipant>
 */
class SprintParticipantFactory extends Factory
{
    protected $model = SprintParticipant::class;

    /**
     * @return array<string, mixed>
     */
    public function definition(): array
    {
        return [
            'sprint_id' => Sprint::factory(),
            'person_id' => Person::factory(),
            'carry_forward_balance' => 0,
            'active_reward_id' => null,
            'display_order' => 0,
        ];
    }
}
