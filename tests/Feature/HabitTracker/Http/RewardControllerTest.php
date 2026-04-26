<?php

namespace Tests\Feature\HabitTracker\Http;

use App\Models\Habit;
use App\Models\Person;
use App\Models\Reward;
use App\Models\Sprint;
use App\Models\SprintParticipant;
use App\Models\Tick;
use Illuminate\Foundation\Testing\LazilyRefreshDatabase;
use Illuminate\Support\Carbon;
use Tests\TestCase;

class RewardControllerTest extends TestCase
{
    use LazilyRefreshDatabase;

    public function test_store_creates_reward(): void
    {
        $person = Person::factory()->create();

        $this->from(route('settings.index'))
            ->post(route('rewards.store', $person), [
                'name' => 'Lego',
                'cost' => 250,
            ])
            ->assertRedirect(route('settings.index'))
            ->assertSessionHas('success', 'Reward set');

        $this->assertDatabaseHas('rewards', [
            'person_id' => $person->id,
            'name' => 'Lego',
            'cost' => 250,
        ]);
    }

    public function test_store_rejects_when_active_reward_exists(): void
    {
        $person = Person::factory()->create();
        Reward::factory()->for($person)->create();

        $this->from(route('settings.index'))
            ->post(route('rewards.store', $person), [
                'name' => 'New',
                'cost' => 100,
            ])
            ->assertSessionHasErrors('reward');
    }

    public function test_store_validates_name_and_cost(): void
    {
        $person = Person::factory()->create();

        $this->from(route('settings.index'))
            ->post(route('rewards.store', $person), [
                'name' => '',
                'cost' => 0,
            ])
            ->assertSessionHasErrors(['name', 'cost']);
    }

    public function test_update_renames_reward(): void
    {
        $person = Person::factory()->create();
        $reward = Reward::factory()->for($person)->create(['name' => 'Old']);

        $this->from(route('settings.index'))
            ->patch(route('rewards.update', $reward), ['name' => 'New'])
            ->assertRedirect(route('settings.index'))
            ->assertSessionHas('success', 'Reward updated');

        $this->assertSame('New', $reward->refresh()->name);
    }

    public function test_update_rejects_cost_edit_on_achieved(): void
    {
        $person = Person::factory()->create();
        $reward = Reward::factory()->for($person)->achieved()->create(['cost' => 50]);

        $this->from(route('settings.index'))
            ->patch(route('rewards.update', $reward), ['cost' => 100])
            ->assertSessionHasErrors('cost');
    }

    public function test_update_flashes_achievement_when_cost_lowered_below_balance(): void
    {
        Carbon::setTestNow('2026-04-26 10:00:00');

        $person = Person::factory()->create(['name' => 'Alice']);
        $sprint = Sprint::factory()->create([
            'start_date' => '2026-04-19',
            'end_date' => '2026-05-16',
            'started_at' => '2026-04-19 00:00:00',
        ]);
        $reward = Reward::factory()->for($person)->create(['name' => 'Lego', 'cost' => 100]);
        $participant = SprintParticipant::factory()
            ->for($sprint)
            ->for($person)
            ->create(['active_reward_id' => $reward->id]);
        $habit = Habit::factory()->for($participant, 'sprintParticipant')->create();
        Tick::create(['habit_id' => $habit->id, 'tick_date' => '2026-04-20']);
        Tick::create(['habit_id' => $habit->id, 'tick_date' => '2026-04-21']);

        $this->from(route('settings.index'))
            ->patch(route('rewards.update', $reward), ['cost' => 2])
            ->assertRedirect(route('settings.index'))
            ->assertSessionHas('achievement', fn ($a) => $a['person_id'] === $person->id
                && $a['reward_id'] === $reward->id
                && $a['reward_name'] === 'Lego'
            );

        $this->assertNotNull($reward->refresh()->achieved_at);
    }

    public function test_destroy_deletes_unachieved_reward(): void
    {
        $person = Person::factory()->create();
        $reward = Reward::factory()->for($person)->create();

        $this->from(route('settings.index'))
            ->delete(route('rewards.destroy', $reward))
            ->assertRedirect(route('settings.index'))
            ->assertSessionHas('success', 'Reward deleted');

        $this->assertDatabaseMissing('rewards', ['id' => $reward->id]);
    }

    public function test_destroy_rejects_achieved_reward(): void
    {
        $person = Person::factory()->create();
        $reward = Reward::factory()->for($person)->achieved()->create();

        $this->from(route('settings.index'))
            ->delete(route('rewards.destroy', $reward))
            ->assertSessionHasErrors('reward');
    }

    public function test_destroy_rejects_reward_linked_to_active_participant(): void
    {
        $person = Person::factory()->create();
        $reward = Reward::factory()->for($person)->create();
        $sprint = Sprint::factory()->create();
        SprintParticipant::factory()
            ->for($sprint)
            ->for($person)
            ->create(['active_reward_id' => $reward->id]);

        $this->from(route('settings.index'))
            ->delete(route('rewards.destroy', $reward))
            ->assertSessionHasErrors('reward');
    }
}
