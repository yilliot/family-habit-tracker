<?php

namespace Tests\Feature\HabitTracker;

use App\Actions\HabitTracker\CreatePerson;
use App\Actions\HabitTracker\SoftDeletePerson;
use App\Actions\HabitTracker\UpdatePerson;
use App\Models\Habit;
use App\Models\Person;
use App\Models\Sprint;
use App\Models\SprintParticipant;
use App\Models\Tick;
use Illuminate\Foundation\Testing\LazilyRefreshDatabase;
use Tests\TestCase;

class PersonTest extends TestCase
{
    use LazilyRefreshDatabase;

    public function test_create_person_assigns_next_display_order(): void
    {
        Person::factory()->create(['display_order' => 5]);

        $person = app(CreatePerson::class)->execute('Charlie');

        $this->assertSame('Charlie', $person->name);
        $this->assertSame(6, $person->display_order);
    }

    public function test_update_person(): void
    {
        $person = Person::factory()->create(['name' => 'Bob', 'display_order' => 1]);

        $updated = app(UpdatePerson::class)->execute($person, [
            'name' => 'Robert',
            'display_order' => 7,
        ]);

        $this->assertSame('Robert', $updated->name);
        $this->assertSame(7, $updated->display_order);
    }

    public function test_soft_delete_person_preserves_history(): void
    {
        $person = Person::factory()->create();
        $sprint = Sprint::factory()->archived()->create();
        $participant = SprintParticipant::factory()->for($sprint)->for($person)->create();
        $habit = Habit::factory()->for($participant, 'sprintParticipant')->create();
        Tick::create(['habit_id' => $habit->id, 'tick_date' => '2026-03-01']);

        app(SoftDeletePerson::class)->execute($person);

        $this->assertSoftDeleted('people', ['id' => $person->id]);
        // Past archived participant rows / habits / ticks are preserved.
        $this->assertSame(1, SprintParticipant::where('id', $participant->id)->count());
        $this->assertSame(1, Habit::where('id', $habit->id)->count());
        $this->assertSame(1, Tick::where('habit_id', $habit->id)->count());
    }

    public function test_soft_delete_person_detaches_from_active_sprint(): void
    {
        $person = Person::factory()->create();
        $sprint = Sprint::factory()->create();
        $participant = SprintParticipant::factory()->for($sprint)->for($person)->create();
        $habit = Habit::factory()->for($participant, 'sprintParticipant')->create();
        Tick::create(['habit_id' => $habit->id, 'tick_date' => '2026-04-25']);

        app(SoftDeletePerson::class)->execute($person);

        $this->assertSoftDeleted('people', ['id' => $person->id]);
        $this->assertSame(0, SprintParticipant::where('id', $participant->id)->count());
        // Habit soft-deleted, but ticks still exist.
        $this->assertSoftDeleted('habits', ['id' => $habit->id]);
        $this->assertSame(1, Tick::where('habit_id', $habit->id)->count());
    }
}
