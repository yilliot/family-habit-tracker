<?php

namespace Tests\Feature\HabitTracker\Http;

use App\Models\Habit;
use App\Models\Person;
use App\Models\Sprint;
use App\Models\SprintParticipant;
use Illuminate\Foundation\Testing\LazilyRefreshDatabase;
use Tests\TestCase;

class HabitControllerTest extends TestCase
{
    use LazilyRefreshDatabase;

    private function buildActiveParticipant(): SprintParticipant
    {
        $sprint = Sprint::factory()->create();
        $person = Person::factory()->create();

        return SprintParticipant::factory()->for($sprint)->for($person)->create();
    }

    public function test_store_adds_habit(): void
    {
        $participant = $this->buildActiveParticipant();

        $this->from(route('settings.index'))
            ->post(route('habits.store', $participant), ['name' => 'Read'])
            ->assertRedirect(route('settings.index'))
            ->assertSessionHas('success', 'Habit added');

        $this->assertDatabaseHas('habits', [
            'sprint_participant_id' => $participant->id,
            'name' => 'Read',
        ]);
    }

    public function test_store_validation_rejects_blank_name(): void
    {
        $participant = $this->buildActiveParticipant();

        $this->from(route('settings.index'))
            ->post(route('habits.store', $participant), ['name' => ''])
            ->assertSessionHasErrors('name');
    }

    public function test_store_blocked_on_archived_sprint(): void
    {
        $participant = $this->buildActiveParticipant();
        $participant->sprint->update(['status' => Sprint::STATUS_ARCHIVED]);

        $this->from(route('settings.index'))
            ->post(route('habits.store', $participant), ['name' => 'Read'])
            ->assertSessionHasErrors('sprint');
    }

    public function test_update_changes_name(): void
    {
        $participant = $this->buildActiveParticipant();
        $habit = Habit::factory()->for($participant, 'sprintParticipant')->create(['name' => 'Old']);

        $this->from(route('settings.index'))
            ->patch(route('habits.update', $habit), ['name' => 'New'])
            ->assertRedirect(route('settings.index'))
            ->assertSessionHas('success', 'Habit updated');

        $this->assertSame('New', $habit->refresh()->name);
    }

    public function test_destroy_soft_deletes_habit(): void
    {
        $participant = $this->buildActiveParticipant();
        $habit = Habit::factory()->for($participant, 'sprintParticipant')->create();

        $this->from(route('settings.index'))
            ->delete(route('habits.destroy', $habit))
            ->assertRedirect(route('settings.index'))
            ->assertSessionHas('success', 'Habit removed');

        $this->assertSoftDeleted('habits', ['id' => $habit->id]);
    }
}
