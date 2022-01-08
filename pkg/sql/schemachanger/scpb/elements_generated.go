// Copyright 2021 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

// Code generated by element_generator.go. DO NOT EDIT.

package scpb

type ElementStatusIterator interface {
	ForEachElementStatus(fn func(status, targetStatus Status, elem Element))
}


func (e Column) element() {}

// ForEachColumn iterates over nodes of type Column.
func ForEachColumn(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *Column),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*Column); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}

func (e PrimaryIndex) element() {}

// ForEachPrimaryIndex iterates over nodes of type PrimaryIndex.
func ForEachPrimaryIndex(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *PrimaryIndex),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*PrimaryIndex); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}

func (e SecondaryIndex) element() {}

// ForEachSecondaryIndex iterates over nodes of type SecondaryIndex.
func ForEachSecondaryIndex(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *SecondaryIndex),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*SecondaryIndex); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}

func (e SequenceDependency) element() {}

// ForEachSequenceDependency iterates over nodes of type SequenceDependency.
func ForEachSequenceDependency(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *SequenceDependency),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*SequenceDependency); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}

func (e UniqueConstraint) element() {}

// ForEachUniqueConstraint iterates over nodes of type UniqueConstraint.
func ForEachUniqueConstraint(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *UniqueConstraint),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*UniqueConstraint); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}

func (e CheckConstraint) element() {}

// ForEachCheckConstraint iterates over nodes of type CheckConstraint.
func ForEachCheckConstraint(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *CheckConstraint),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*CheckConstraint); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}

func (e Sequence) element() {}

// ForEachSequence iterates over nodes of type Sequence.
func ForEachSequence(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *Sequence),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*Sequence); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}

func (e DefaultExpression) element() {}

// ForEachDefaultExpression iterates over nodes of type DefaultExpression.
func ForEachDefaultExpression(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *DefaultExpression),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*DefaultExpression); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}

func (e View) element() {}

// ForEachView iterates over nodes of type View.
func ForEachView(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *View),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*View); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}

func (e Table) element() {}

// ForEachTable iterates over nodes of type Table.
func ForEachTable(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *Table),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*Table); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}

func (e ForeignKey) element() {}

// ForEachForeignKey iterates over nodes of type ForeignKey.
func ForEachForeignKey(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *ForeignKey),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*ForeignKey); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}

func (e ForeignKeyBackReference) element() {}

// ForEachForeignKeyBackReference iterates over nodes of type ForeignKeyBackReference.
func ForEachForeignKeyBackReference(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *ForeignKeyBackReference),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*ForeignKeyBackReference); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}

func (e RelationDependedOnBy) element() {}

// ForEachRelationDependedOnBy iterates over nodes of type RelationDependedOnBy.
func ForEachRelationDependedOnBy(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *RelationDependedOnBy),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*RelationDependedOnBy); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}

func (e SequenceOwnedBy) element() {}

// ForEachSequenceOwnedBy iterates over nodes of type SequenceOwnedBy.
func ForEachSequenceOwnedBy(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *SequenceOwnedBy),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*SequenceOwnedBy); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}

func (e Type) element() {}

// ForEachType iterates over nodes of type Type.
func ForEachType(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *Type),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*Type); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}

func (e Schema) element() {}

// ForEachSchema iterates over nodes of type Schema.
func ForEachSchema(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *Schema),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*Schema); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}

func (e Database) element() {}

// ForEachDatabase iterates over nodes of type Database.
func ForEachDatabase(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *Database),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*Database); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}

func (e Partitioning) element() {}

// ForEachPartitioning iterates over nodes of type Partitioning.
func ForEachPartitioning(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *Partitioning),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*Partitioning); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}

func (e Namespace) element() {}

// ForEachNamespace iterates over nodes of type Namespace.
func ForEachNamespace(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *Namespace),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*Namespace); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}

func (e Owner) element() {}

// ForEachOwner iterates over nodes of type Owner.
func ForEachOwner(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *Owner),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*Owner); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}

func (e UserPrivileges) element() {}

// ForEachUserPrivileges iterates over nodes of type UserPrivileges.
func ForEachUserPrivileges(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *UserPrivileges),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*UserPrivileges); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}

func (e ColumnName) element() {}

// ForEachColumnName iterates over nodes of type ColumnName.
func ForEachColumnName(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *ColumnName),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*ColumnName); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}

func (e Locality) element() {}

// ForEachLocality iterates over nodes of type Locality.
func ForEachLocality(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *Locality),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*Locality); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}

func (e IndexName) element() {}

// ForEachIndexName iterates over nodes of type IndexName.
func ForEachIndexName(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *IndexName),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*IndexName); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}

func (e ConstraintName) element() {}

// ForEachConstraintName iterates over nodes of type ConstraintName.
func ForEachConstraintName(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *ConstraintName),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*ConstraintName); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}

func (e DefaultExprTypeReference) element() {}

// ForEachDefaultExprTypeReference iterates over nodes of type DefaultExprTypeReference.
func ForEachDefaultExprTypeReference(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *DefaultExprTypeReference),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*DefaultExprTypeReference); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}

func (e OnUpdateExprTypeReference) element() {}

// ForEachOnUpdateExprTypeReference iterates over nodes of type OnUpdateExprTypeReference.
func ForEachOnUpdateExprTypeReference(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *OnUpdateExprTypeReference),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*OnUpdateExprTypeReference); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}

func (e ComputedExprTypeReference) element() {}

// ForEachComputedExprTypeReference iterates over nodes of type ComputedExprTypeReference.
func ForEachComputedExprTypeReference(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *ComputedExprTypeReference),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*ComputedExprTypeReference); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}

func (e ViewDependsOnType) element() {}

// ForEachViewDependsOnType iterates over nodes of type ViewDependsOnType.
func ForEachViewDependsOnType(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *ViewDependsOnType),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*ViewDependsOnType); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}

func (e ColumnTypeReference) element() {}

// ForEachColumnTypeReference iterates over nodes of type ColumnTypeReference.
func ForEachColumnTypeReference(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *ColumnTypeReference),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*ColumnTypeReference); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}

func (e DatabaseSchemaEntry) element() {}

// ForEachDatabaseSchemaEntry iterates over nodes of type DatabaseSchemaEntry.
func ForEachDatabaseSchemaEntry(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *DatabaseSchemaEntry),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*DatabaseSchemaEntry); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}

func (e CheckConstraintTypeReference) element() {}

// ForEachCheckConstraintTypeReference iterates over nodes of type CheckConstraintTypeReference.
func ForEachCheckConstraintTypeReference(
	b ElementStatusIterator, elementFunc func(status, targetStatus Status, element *CheckConstraintTypeReference),
) {
	b.ForEachElementStatus(func(status, targetStatus Status, elem Element) {
		if e, ok := elem.(*CheckConstraintTypeReference); ok {
			elementFunc(status, targetStatus, e)
		}
	})
}