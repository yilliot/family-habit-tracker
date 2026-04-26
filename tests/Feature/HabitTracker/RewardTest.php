<?php

namespace Tests\Feature\HabitTracker;

use App\Actions\HabitTracker\CreateReward;
use App\Actions\HabitTracker\DeleteReward;
use App\Actions\HabitTracker\UpdateReward;
use App\Models\Habit;
use App\Models\Person;
use App\Models\Reward;
use App\Models\Sprint;
use App\Models\SprintParticipant;
use App\Models\Tick;
use Illuminate\Foundation\Testing\LazilyRefreshDatabase;
use Illuminate\Support\Carbon;
use Illuminate\Validation\ValidationException;
use Tests\TestCase;

class RewardTest extends TestCase
{
    use LazilyRefreshDatabase;

    public function test_create_reward_for_person_without_active_one(): void
    {
        $person = Person::factory()->create();

        $reward = app(CreateReward::class)->execute($person, 'Lego', 250);

        $this->assertSame('Lego', $reward->name);
        $this->assertSame(250, $reward->cost);
    }

    public function test_create_reward_rejected_when_person_already_has_active_one(): void
    {
        $person = Person::factory()->create();
        Reward::factory()->for($person)->create();

        $this->expectException(ValidationException::class);
        app(CreateReward::class)->execute($person, 'Another', 100);
    }

    public function test_update_reward_name_when_achieved(): void
    {
        $person = Person::factory()->create();
        $reward = Reward::factory()->for($person)->achieved()->create(['name' => 'Old', 'cost' => 50]);

        $updated = app(UpdateReward::class)->execute($reward, ['name' => 'New Label']);

        $this->assertSame('New Label', $updated->name);
        $this->assertSame(50, $updated->cost);
    }

    public function test_update_cost_blocked_on_achieved_reward(): void
    {
        $person = Person::factory()->create();
        $reward = Reward::factory()->for($person)->achieved()->create(['cost' => 50]);

        $this->expectException(ValidationException::class);
        app(UpdateReward::class)->execute($reward, ['cost' => 30]);
    }

    public function test_lowering_cost_marks_reward_achieved_when_balance_already_high(): void
    {
        Carbon::setTestNow('2026-04-26 10:00:00');

        $person = Person::factory()->create();
        $sprint = Sprint::factory()->create([
            'start_date' => '2026-04-19',
            'end_date' => '2026-05-16',
            'started_at' => '2026-04-19 00:00:00',
        ]);
        $reward = Reward::factory()->for($person)->create(['cost' => 100]);
        $participant = SprintParticipant::factory()
            ->for($sprint)
            ->for($person)
            ->create(['active_reward_id' => $reward->id, 'carry_forward_balance' => 0]);
        $habit = Habit::factory()->for($participant, 'sprintParticipant')->create();

        // Earn 5 ticks.
        foreach (range(0, 4) as $i) {
            Tick::create([
                'habit_id' => $habit->id,
                'tick_date' => Carbon::parse('2026-04-20')->addDays($i)->toDateString(),
            ]);
        }

        $updated = app(UpdateReward::class)->execute($reward, ['cost' => 5]);

        $this->assertNotNull($updated->achieved_at);
    }

    public function test_delete_unachieved_reward_not_linked_to_active_sprint(): void
    {
        $person = Person::factory()->create();
        $reward = Reward::factory()->for($person)->create();

        app(DeleteReward::class)->execute($reward);

        $this->assertNull(Reward::find($reward->id));
    }

    public function test_delete_blocked_for_achieved_reward(): void
    {
        $person = Person::factory()->create();
        $reward = Reward::factory()->for($person)->achieved()->create();

        $this->expectException(ValidationException::class);
        app(DeleteReward::class)->execute($reward);
    }

    public function test_delete_blocked_when_linked_to_active_sprint_participant(): void
    {
        $person = Person::factory()->create();
        $sprint = Sprint::factory()->create();
        $reward = Reward::factory()->for($person)->create();
        SprintParticipant::factory()
            ->for($sprint)
            ->for($person)
            ->create(['active_reward_id' => $reward->id]);

        $this->expectException(ValidationException::class);
        app(DeleteReward::class)->execute($reward);
    }
}
