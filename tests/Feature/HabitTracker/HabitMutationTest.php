<?php

namespace Tests\Feature\HabitTracker;

use App\Actions\HabitTracker\CreateHabit;
use App\Actions\HabitTracker\SoftDeleteHabit;
use App\Actions\HabitTracker\UpdateHabit;
use App\Models\Habit;
use App\Models\Person;
use App\Models\Sprint;
use App\Models\SprintParticipant;
use App\Models\Tick;
use Illuminate\Foundation\Testing\LazilyRefreshDatabase;
use Illuminate\Validation\ValidationException;
use Tests\TestCase;

class HabitMutationTest extends TestCase
{
    use LazilyRefreshDatabase;

    public function test_create_habit_on_active_sprint(): void
    {
        $participant = $this->makeParticipant();

        $habit = app(CreateHabit::class)->execute($participant, 'Read');

        $this->assertSame('Read', $habit->name);
        $this->assertSame($participant->id, $habit->sprint_participant_id);
    }

    public function test_create_habit_blocked_on_archived_sprint(): void
    {
        $participant = $this->makeParticipant(archived: true);

        $this->expectException(ValidationException::class);
        app(CreateHabit::class)->execute($participant, 'Read');
    }

    public function test_update_habit_changes_name(): void
    {
        $participant = $this->makeParticipant();
        $habit = Habit::factory()->for($participant, 'sprintParticipant')->create(['name' => 'Old']);

        $updated = app(UpdateHabit::class)->execute($habit, ['name' => 'New']);

        $this->assertSame('New', $updated->name);
    }

    public function test_update_habit_blocked_on_archived_sprint(): void
    {
        $participant = $this->makeParticipant(archived: true);
        $habit = Habit::factory()->for($participant, 'sprintParticipant')->create();

        $this->expectException(ValidationException::class);
        app(UpdateHabit::class)->execute($habit, ['name' => 'New']);
    }

    public function test_soft_delete_habit_preserves_ticks(): void
    {
        $participant = $this->makeParticipant();
        $habit = Habit::factory()->for($participant, 'sprintParticipant')->create();
        Tick::create(['habit_id' => $habit->id, 'tick_date' => '2026-04-25']);

        app(SoftDeleteHabit::class)->execute($habit);

        $this->assertSoftDeleted('habits', ['id' => $habit->id]);
        $this->assertSame(1, Tick::where('habit_id', $habit->id)->count());
    }

    public function test_soft_delete_habit_blocked_on_archived_sprint(): void
    {
        $participant = $this->makeParticipant(archived: true);
        $habit = Habit::factory()->for($participant, 'sprintParticipant')->create();

        $this->expectException(ValidationException::class);
        app(SoftDeleteHabit::class)->execute($habit);
    }

    private function makeParticipant(bool $archived = false): SprintParticipant
    {
        $person = Person::factory()->create();
        $sprint = $archived
            ? Sprint::factory()->archived()->create()
            : Sprint::factory()->create();

        return SprintParticipant::factory()->for($sprint)->for($person)->create();
    }
}
