<?php

namespace Tests\Feature\HabitTracker\Http;

use App\Models\Person;
use App\Models\Reward;
use App\Models\Sprint;
use App\Models\SprintParticipant;
use Illuminate\Foundation\Testing\LazilyRefreshDatabase;
use Tests\TestCase;

class SprintParticipantControllerTest extends TestCase
{
    use LazilyRefreshDatabase;

    public function test_store_adds_participant(): void
    {
        $sprint = Sprint::factory()->create();
        $person = Person::factory()->create();

        $this->from(route('settings.index'))
            ->post(route('sprint-participants.store', $sprint), [
                'person_id' => $person->id,
                'habits' => [['name' => 'Hydrate']],
                'reward' => ['mode' => 'new', 'name' => 'Treat', 'cost' => 20],
            ])
            ->assertRedirect(route('settings.index'))
            ->assertSessionHas('success', 'Participant added');

        $this->assertDatabaseHas('sprint_participants', [
            'sprint_id' => $sprint->id,
            'person_id' => $person->id,
        ]);
    }

    public function test_store_rejects_archived_sprint(): void
    {
        $sprint = Sprint::factory()->archived()->create();
        $person = Person::factory()->create();

        $this->from(route('settings.index'))
            ->post(route('sprint-participants.store', $sprint), [
                'person_id' => $person->id,
            ])
            ->assertSessionHasErrors('sprint');
    }

    public function test_store_rejects_missing_person(): void
    {
        $sprint = Sprint::factory()->create();

        $this->from(route('settings.index'))
            ->post(route('sprint-participants.store', $sprint), [
                'person_id' => 999999,
            ])
            ->assertSessionHasErrors('person_id');
    }

    public function test_update_reward_swaps_active_reward(): void
    {
        $sprint = Sprint::factory()->create();
        $person = Person::factory()->create();
        $reward = Reward::factory()->for($person)->create();
        $participant = SprintParticipant::factory()
            ->for($sprint)
            ->for($person)
            ->create(['active_reward_id' => null]);

        $this->from(route('settings.index'))
            ->post(route('sprint-participants.update-reward', $participant), [
                'reward_id' => $reward->id,
            ])
            ->assertRedirect(route('settings.index'))
            ->assertSessionHas('success', 'Active reward updated');

        $this->assertSame($reward->id, $participant->refresh()->active_reward_id);
    }

    public function test_update_reward_rejects_other_persons_reward(): void
    {
        $sprint = Sprint::factory()->create();
        $person = Person::factory()->create();
        $other = Person::factory()->create();
        $reward = Reward::factory()->for($other)->create();
        $participant = SprintParticipant::factory()
            ->for($sprint)
            ->for($person)
            ->create();

        $this->from(route('settings.index'))
            ->post(route('sprint-participants.update-reward', $participant), [
                'reward_id' => $reward->id,
            ])
            ->assertSessionHasErrors('reward_id');
    }

    public function test_update_reward_rejects_achieved_reward(): void
    {
        $sprint = Sprint::factory()->create();
        $person = Person::factory()->create();
        $reward = Reward::factory()->for($person)->achieved()->create();
        $participant = SprintParticipant::factory()
            ->for($sprint)
            ->for($person)
            ->create();

        $this->from(route('settings.index'))
            ->post(route('sprint-participants.update-reward', $participant), [
                'reward_id' => $reward->id,
            ])
            ->assertSessionHasErrors('reward_id');
    }
}
